package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-acme/lego/v3/certcrypto"
	"github.com/go-acme/lego/v3/certificate"
	"github.com/go-acme/lego/v3/challenge/http01"
	"github.com/go-acme/lego/v3/challenge/tlsalpn01"
	"github.com/go-acme/lego/v3/lego"
	"github.com/go-acme/lego/v3/registration"
)

// You'll need a user or account type that implements acme.User
type MyUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *MyUser) GetEmail() string {
	return u.Email
}
func (u MyUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *MyUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func getCertificatefromACME(storepath string, cfg Config) error {
	// Check path existence
	if _, err := os.Stat(storepath); os.IsNotExist(err) {
		//fmt.Printf("Couldn't open %s", storepath)
		return err
	}

	// Create a user. New accounts need an email and private key to start.
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	myUser := MyUser{
		Email: cfg.Acme.Useremail,
		key:   privateKey,
	}

	config := lego.NewConfig(&myUser)

	// This CA URL is configured for a local dev instance of Boulder running in Docker in a VM.
	config.CADirURL = cfg.Acme.Cadirurl
	config.Certificate.KeyType = certcrypto.RSA2048

	// A client facilitates communication with the CA server.
	client, err := lego.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// We specify an http port of 5002 and an tls port of 5001 on all interfaces
	// because we aren't running as root and can't bind a listener to port 80 and 443
	// (used later when we attempt to pass challenges). Keep in mind that you still
	// need to proxy challenge traffic to port 5002 and 5001.
	//err = client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "5002"))
	err = client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "80"))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer("", "5001"))
	//err = client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer("", "443"))
	if err != nil {
		log.Fatal(err)
	}

	// New users will need to register
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		log.Fatal(err)
	}
	myUser.Registration = reg

	request := certificate.ObtainRequest{
		Domains: []string{cfg.Acme.Hostname},
		Bundle:  false,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		log.Fatal(err)
	}

	// Each certificate comes back with the cert bytes, the bytes of the client's
	// private key, and a certificate URL. SAVE THESE TO DISK.
	//fmt.Printf("%#v\n", certificates)
	//fmt.Print(certificates.Certificate)
	absolutePath, _ := filepath.Abs(storepath)
	err = ioutil.WriteFile(absolutePath+"/privatekey.pem", certificates.PrivateKey, 0644)
	err = ioutil.WriteFile(absolutePath+"/certificate.pem", certificates.Certificate, 0644)
	err = ioutil.WriteFile(absolutePath+"/issuer.pem", certificates.IssuerCertificate, 0644)

	if err != nil {
		panic(err)
	}

	rebopCertificate := rebopCertificate{
		hostname,
		"",
		ipaddress,
		"certificate.pem",
		absolutePath + "/certificate.pem",
		string(certificates.Certificate),
		time.Now().UTC().Format("2006-01-02T15:04:05z"),
		"local",
	}
	rebopCertificates := make(rebopCertificates, 0)
	rebopCertificates = append(rebopCertificates, rebopCertificate)
	certificateJSON, err := json.Marshal(rebopCertificates)
	if err != nil {
		return err
	}

	err = rebopSend(certificateJSON, "ACME", cfg)
	if err != nil {
		return err
	}
	fmt.Println("reBop renew Completed:", absolutePath+"/certificate.pem", "\nCertificate successfully sent to rebop")
	return nil
	// ... all done.
}
