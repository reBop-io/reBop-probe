package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ks "github.com/pavlo-v-chernykh/keystore-go/v4"
	"github.com/rebop-io/reBop-probe/log"
	"software.sslmate.com/src/go-pkcs12"
)

type rebopCertificate struct {
	Hostname  string   `json:"hostname"`
	Port      string   `json:"port"`
	Ipaddress []string `json:"ipaddress"`
	Filename  string   `json:"filename"`
	Path      string   `json:"path"`
	//privatekeypath string `json:"privatekeypath"`
	Certificate string `json:"certificate"`
	Date        string `json:"date"`
	Probe       string `json:"probe"`
}

type rebopCertificates []rebopCertificate

type fsEntry struct {
	path string
	f    os.FileInfo
}

func rebopScan(rootPath string) (int, []byte, error) {
	log.Infof("Starting scan of: %s", rootPath)
	// 1. Add context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Check if path exists
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return 0, nil, fmt.Errorf("path does not exist: %s", rootPath)
	}

	// 3. Configure concurrency
	numWorkers := runtime.NumCPU()
	paths := make(chan fsEntry, numWorkers*10) // Buffer based on worker count
	errs := make(chan error, numWorkers*10)    // Buffer error channel
	certs := make(chan *rebopCertificate, numWorkers*10)
	done := make(chan struct{})

	// Create separate WaitGroups for file walker and workers
	var fileWalkerWg, workerWg sync.WaitGroup

	// 4. Start file walker
	fileWalkerWg.Add(1)
	go func() {
		defer fileWalkerWg.Done()
		if err := filepath.Walk(rootPath, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				select {
				case errs <- err:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}

			// Early filtering by extension
			extension := filepath.Ext(path)
			if !stringInSlice(extension, ext) {
				return nil
			}

			if !f.IsDir() {
				select {
				case paths <- fsEntry{path: path, f: f}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		}); err != nil {
			select {
			case errs <- err:
			case <-ctx.Done():
			}
		}
	}()

	// 5. Start worker pool
	for i := 0; i < numWorkers; i++ {
		workerWg.Add(1)
		go func() {
			certWorker(ctx, paths, errs, certs, &workerWg)
		}()
	}

	// 6. Close paths channel when the file walker is done
	go func() {
		fileWalkerWg.Wait()
		close(paths) // Close paths to signal workers that no more entries are coming
	}()

	// 7. Close certs and done when all workers are finished
	go func() {
		workerWg.Wait()
		close(certs)
		close(done)
	}()

	// 7. Process results with progress reporting
	var (
		rebopCerts rebopCertificates
		lastUpdate = time.Now()
		updateFreq = 100 * time.Millisecond
	)

	// Track if we've seen the done signal
	var doneReceived bool

	for {
		select {
		case <-ctx.Done():
			return 0, nil, ctx.Err()

		case cert, ok := <-certs:
			if !ok {
				certs = nil
				// If we've already received the done signal, we can exit now
				if doneReceived {
					break
				}
				continue
			}
			rebopCerts = append(rebopCerts, *cert)

		case err := <-errs:
			if debug && err != nil {
				log.Infof("Error processing file: %v", err)
			}

		case <-time.After(updateFreq):
			if time.Since(lastUpdate) > time.Second {
				fmt.Fprintf(os.Stderr, "\rParsed %d files, found %d certificates",
					atomic.LoadInt64(&parsedCount),
					len(rebopCerts),
				)
				lastUpdate = time.Now()
			}

		case <-done:
			doneReceived = true
			// If certs is already closed, we can proceed to marshal and return
			if certs == nil {
				certJSON, err := json.Marshal(rebopCerts)
				if err != nil {
					return 0, nil, fmt.Errorf("error marshaling certificates: %w", err)
				}

				log.Infof("reBop scan completed in %s", time.Since(start))
				log.Infof("Parsed: %d files", parsedCount)
				log.Infof("Found: %d new certificates, %d known files, %d errors",
					validCount, knownCount, errorCount)

				return len(rebopCerts), certJSON, nil
			}
		}

		// Exit condition: both certs channel is closed and we've received done signal
		if certs == nil && doneReceived {
			certJSON, err := json.Marshal(rebopCerts)
			if err != nil {
				return 0, nil, fmt.Errorf("error marshaling certificates: %w", err)
			}

			log.Infof("reBop scan completed in %s", time.Since(start))
			log.Infof("Parsed: %d files", parsedCount)
			log.Infof("Found: %d new certificates, %d known files, %d errors",
				validCount, knownCount, errorCount)

			return len(rebopCerts), certJSON, nil
		}
	}

	return 0, nil, nil
}

func parseHostForCertFiles(pathS string, paths chan<- fsEntry, errs chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	filepath.Walk(pathS, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			select {
			case errs <- err:
			case <-context.Background().Done():
				return context.Canceled
			}
			return nil
		}
		if !f.IsDir() {
			absolutePath, absErr := filepath.Abs(path)
			if absErr != nil {
				select {
				case errs <- fmt.Errorf("error getting absolute path for %s: %w", path, absErr):
				case <-context.Background().Done():
					return context.Canceled
				}
				return nil
			}
			select {
			case paths <- fsEntry{path: absolutePath, f: f}:
			case <-context.Background().Done():
				return context.Canceled
			}
		}
		return nil
	})
}

func certWorker(ctx context.Context, entries <-chan fsEntry, errs chan<- error, certs chan<- *rebopCertificate, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case entry, ok := <-entries:
			if !ok {
				return
			}
			cert, err := parseEntry(entry)
			if err != nil {
				select {
				case errs <- fmt.Errorf("error parsing %s: %w", entry.path, err):
				case <-ctx.Done():
					return
				}
				continue
			}
			if cert != nil {
				select {
				case certs <- cert:
				case <-ctx.Done():
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func parseEntry(entry fsEntry) (*rebopCertificate, error) {
	var cert string

	// Skip if not a certificate file
	if !stringInSlice(filepath.Ext(entry.path), ext) {
		return nil, nil
	}

	mutex.Lock()
	atomic.AddInt64(&parsedCount, 1)
	mutex.Unlock()

	dat, err := os.ReadFile(entry.path)
	if err != nil {
		return nil, err
	}

	if len(dat) == 0 {
		return nil, fmt.Errorf("empty file")
	}

	switch {
	case isJKSFile(dat):
		// Try with empty password first, then common passwords
		passwords := []string{"", "changeit", "password", "changeme", "secret", "123456"}
		var lastErr error
		for _, pass := range passwords {
			cert, err = parseJKS(dat, pass)
			if err == nil {
				break
			}
			lastErr = err
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse JKS: %v", lastErr)
		}

	case isPKCS12File(dat):
		// Try with empty password first, then common passwords
		passwords := []string{"", "changeit", "password", "changeme", "secret"}
		var lastErr error
		for _, pass := range passwords {
			cert, err = parsePKCS12(dat, pass)
			if err == nil {
				break
			}
			lastErr = err
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS12: %v", lastErr)
		}

	default:
		// Handle standard PEM/DER formats
		if strings.Contains(string(dat), ("PRIVATE KEY")) || strings.Contains(string(dat), ("PUBLIC KEY")) {
			mutex.Lock()
			errorCount++
			mutex.Unlock()
			return nil, nil
		} else if !strings.Contains(string(dat), ("-----BEGIN CERTIFICATE-----")) {
			// Handle DER format
			cert = base64.StdEncoding.EncodeToString(dat)
			cert = insertNth(cert, 64)
			cert = "-----BEGIN CERTIFICATE-----\n" + cert + "\n-----END CERTIFICATE-----"
		} else {
			// Handle PEM format
			cert = string(dat)
		}
	}

	// Parse the certificate to ensure it's valid
	block, _ := pem.Decode([]byte(cert))
	if block == nil {
		mutex.Lock()
		errorCount++
		mutex.Unlock()
		if debug {
			fmt.Println("failed to parse PEM file: ", entry.path)
		}
		return nil, nil
	}

	_, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		if strings.Contains(err.Error(), "named curve") {
			if debug {
				fmt.Println(err.Error())
			}
		} else {
			mutex.Lock()
			errorCount++
			mutex.Unlock()
			return nil, fmt.Errorf("failed to parse certificate: %v", err)
		}
	}

	// Calculate hashes for deduplication
	pathhash := sha256.Sum256([]byte(entry.path))
	datahash := sha256.Sum256([]byte(cert))

	mutex.Lock()
	defer mutex.Unlock()

	// Check for duplicates
	if val, ok := hashtable[pathhash]; ok && val == datahash {
		knownCount++
		return nil, nil
	}

	// Add to hashtable and increment counters
	hashtable[pathhash] = datahash
	validCount++

	// Create and return the certificate
	rebopCert := rebopCertificate{
		hostname,
		"",
		ipaddress,
		entry.f.Name(),
		entry.path,
		cert,
		time.Now().UTC().Format("2006-01-02T15:04:05z"),
		"local",
	}

	return &rebopCert, nil
}

// isJKSFile checks if the given data is a JKS file
func isJKSFile(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// JKS files start with a 4-byte magic number: 0xFEEDFEED
	return data[0] == 0xFE && data[1] == 0xED && data[2] == 0xFE && data[3] == 0xED
}

// isPKCS12File checks if the given data is a PKCS12 file
func isPKCS12File(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// PKCS12 files start with a 2-byte version number (0x30 0x82)
	return data[0] == 0x30 && data[1] == 0x82
}

// parseJKS parses a JKS keystore and returns the first certificate found
func parseJKS(data []byte, password string) (string, error) {
	ks := ks.New()
	err := ks.Load(bytes.NewReader(data), []byte(password))
	if err != nil {
		return "", fmt.Errorf("failed to load JKS: %v", err)
	}

	for _, alias := range ks.Aliases() {
		entry, err := ks.GetTrustedCertificateEntry(alias)
		if err != nil {
			continue
		}
		cert := entry.Certificate
		pemCert := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Content,
		})
		return string(pemCert), nil
	}

	return "", fmt.Errorf("no certificate found in JKS")
}

// parsePKCS12 parses a PKCS12 keystore and returns the first certificate found
func parsePKCS12(data []byte, password string) (string, error) {
	privateKey, cert, _, err := pkcs12.DecodeChain(data, password)
	if err != nil {
		return "", fmt.Errorf("failed to decode PKCS12: %v", err)
	}

	// Convert certificate to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	// If there's a private key, we can include it too
	if privateKey != nil {
		keyPEM, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err == nil {
			pemKey := pem.EncodeToMemory(&pem.Block{
				Type:  "PRIVATE KEY",
				Bytes: keyPEM,
			})
			return string(certPEM) + "\n" + string(pemKey), nil
		}
	}

	return string(certPEM), nil
}
