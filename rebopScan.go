package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		//fmt.Printf("Couldn't open %s", rootPath)
		return 0, nil, err
	}
	var wg sync.WaitGroup

	paths := make(chan fsEntry, 1)
	errs := make(chan error, 1)
	certs := make(chan *rebopCertificate, 1)
	done := make(chan bool, 1)

	wg.Add(1)
	go parseHostForCertFiles(rootPath, paths, errs, &wg)
	go certWorker(paths, errs, certs, &wg)
	go func() {
		wg.Wait()
		done <- true
	}()

	rebopCertificates := make(rebopCertificates, 0)

	log.Infof("%s %s started - scanning %s", app.Name, app.Version, rootPath)

	for {
		select {
		case <-done:
			mutex.Lock()
			certificateJSON, err := json.Marshal(rebopCertificates)
			lengh := len(rebopCertificates)
			if err != nil {
				return 0, nil, err
			}
			mutex.Unlock()
			fmt.Printf("\r")
			log.Infof("reBop scan Completed in : %s", time.Since(start))
			log.Infof("Parsed: %d files", parsedCount)
			log.Infof("Found: %d new files with certificate, %d known files and %d files without certificate", validCount, knownCount, errorCount)
			return lengh, certificateJSON, nil
		case cert := <-certs:
			rebopCertificates = append(rebopCertificates, *cert)
		case err := <-errs:
			if debug {
				fmt.Println("error: ", err)
			}
		default:
			fmt.Printf("\rParsed %d files", parsedCount)
		}
	}
}

func parseHostForCertFiles(pathS string, paths chan fsEntry, errs chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	filepath.Walk(pathS, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			errs <- err
			return nil
		}
		if !f.IsDir() {
			absolutePath, _ := filepath.Abs(path)
			paths <- fsEntry{path: absolutePath, f: f}
		}
		return nil
	})
}

func certWorker(entries chan fsEntry, errs chan error, certs chan *rebopCertificate, wg *sync.WaitGroup) {
	for entry := range entries {
		wg.Add(1)
		// too fast, need to save entry value before executing go routine
		entryCopy := entry
		go func() {
			defer wg.Done()
			// FIXME: use a limited pool
			cert, err := parseEntry(entryCopy)
			if err != nil {
				errs <- err
				return
			}
			if cert != nil {
				certs <- cert
			}
		}()
	}
}

func parseEntry(entry fsEntry) (*rebopCertificate, error) {
	var cert string

	// Skip if not a certificate file
	if !stringInSlice(filepath.Ext(entry.path), ext) {
		return nil, nil
	}

	mutex.Lock()
	parsedCount++
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
