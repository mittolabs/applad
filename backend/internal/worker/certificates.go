package worker

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Certificates processes SSL/TLS certificate jobs. Generates self-signed
// certificates for custom domains and stores them on disk for the proxy.
type Certificates struct {
	cfg   *config.Config
	stop  chan struct{}
	queue *queue.Queue
}

func NewCertificates(cfg *config.Config) *Certificates {
	return &Certificates{cfg: cfg, stop: make(chan struct{})}
}

func (w *Certificates) Start() error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	log.Println("certificates worker: listening for jobs")

	ctx := context.Background()
	for {
		select {
		case <-w.stop:
			return nil
		default:
			job, err := w.queue.Pop(ctx, "certificates")
			if err != nil {
				log.Printf("certificates worker: pop error: %v", err)
				continue
			}
			if job == nil {
				continue
			}
			w.process(ctx, job)
		}
	}
}

func (w *Certificates) process(_ context.Context, job *queue.Job) {
	log.Printf("certificates worker: processing job %s", job.ID)

	domain, _ := job.Payload["domain"].(string)
	if domain == "" {
		log.Printf("certificates worker: job %s missing domain", job.ID)
		return
	}

	certDir := filepath.Join(w.cfg.StoragePath, "certs")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		log.Printf("certificates worker: mkdir error: %v", err)
		return
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Printf("certificates worker: key gen error: %v", err)
		return
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: domain},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		DNSNames:     []string{domain},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		log.Printf("certificates worker: cert creation error: %v", err)
		return
	}

	certFile := filepath.Join(certDir, domain+".crt")
	cf, err := os.Create(certFile)
	if err != nil {
		log.Printf("certificates worker: write cert error: %v", err)
		return
	}
	pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	cf.Close()

	keyFile := filepath.Join(certDir, domain+".key")
	kf, err := os.Create(keyFile)
	if err != nil {
		log.Printf("certificates worker: write key error: %v", err)
		return
	}
	keyDER, _ := x509.MarshalECPrivateKey(key)
	pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	kf.Close()

	log.Printf("certificates worker: generated cert for %s → %s", domain, certFile)
}

func (w *Certificates) Stop() { close(w.stop) }
