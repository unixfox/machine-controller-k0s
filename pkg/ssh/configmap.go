package ssh

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	privateKeyDataIndex = "id_rsa"

	secretName = "machine-controller-ssh-key"
)

func EnsureSSHKeypairSecret(client kubernetes.Interface) (*rsa.PrivateKey, error) {
	secret, err := client.CoreV1().Secrets(metav1.NamespaceSystem).Get(secretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			glog.V(4).Info("generating master ssh keypair")
			pk, err := NewPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate ssh keypair: %v", err)
			}

			privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pk)}
			privBuf := bytes.Buffer{}
			err = pem.Encode(&privBuf, privateKeyPEM)
			if err != nil {
				return nil, err
			}

			secret := v1.Secret{}
			secret.Name = secretName
			secret.Type = v1.SecretTypeOpaque

			secret.Data = map[string][]byte{
				privateKeyDataIndex: privBuf.Bytes(),
			}

			_, err = client.CoreV1().Secrets(metav1.NamespaceSystem).Create(&secret)
			if err != nil {
				return nil, err
			}
			return pk, nil
		}
		return nil, err
	}

	return keyFromSecret(secret)
}

func keyFromSecret(secret *v1.Secret) (*rsa.PrivateKey, error) {
	b, exists := secret.Data[privateKeyDataIndex]
	if !exists {
		return nil, fmt.Errorf("key data not found in secret '%s/%s' (secret.data['%s']). remove it and a new one will be created", secret.Namespace, secret.Name, privateKeyDataIndex)
	}
	decoded, _ := pem.Decode(b)
	pk, err := x509.ParsePKCS1PrivateKey(decoded.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return pk, nil
}