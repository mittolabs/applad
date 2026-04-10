package worker

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Certificates processes SSL/TLS certificate jobs.
type Certificates struct {
	cfg   *config.Config
	queue *queue.Queue
}

func NewCertificates(cfg *config.Config) *Certificates {
	return &Certificates{cfg: cfg}
}

func (w *Certificates) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)
	w.queue.StartReaper(ctx, "certificates")

	slog.Info("certificates worker: listening for jobs")

	for {
		receipt, err := w.queue.Pop(ctx, "certificates")
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("certificates worker: shutting down")
				return nil
			}
			slog.Error("certificates worker: pop error", "error", err)
			continue
		}
		if receipt == nil {
			continue
		}
		Heartbeat()
		if processErr := w.process(ctx, receipt.Job); processErr != nil {
			metrics.QueueJobs.Inc("certificates", "failed")
			receipt.Nack()
		} else {
			metrics.QueueJobs.Inc("certificates", "completed")
			receipt.Ack()
		}
	}
}

func (w *Certificates) process(_ context.Context, job *queue.Job) error {
	domain, _ := job.Payload["domain"].(string)
	if domain == "" {
		return nil
	}

	certDir := filepath.Join(w.cfg.StoragePath, "certs")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		return err
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
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
		return err
	}

	certFile := filepath.Join(certDir, domain+".crt")
	cf, err := os.Create(certFile)
	if err != nil {
		return err
	}
	pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}) //nolint:errcheck
	cf.Close()

	keyFile := filepath.Join(certDir, domain+".key")
	kf, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	keyDER, _ := x509.MarshalECPrivateKey(key)
	pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}) //nolint:errcheck
	kf.Close()

	slog.Info("certificates worker: generated cert", "domain", domain, "path", certFile)
	return nil
}
