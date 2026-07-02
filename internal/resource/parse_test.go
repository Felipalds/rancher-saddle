package resource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeYAML(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

const credentialYAML = `
apiVersion: saddle/v1
kind: Credential
metadata:
  name: prod
spec:
  provider: aws
  accessKey: AKIAIOSFODNN7EXAMPLE
  secretKey: supersecret
  region: us-east-1
`

const clusterYAML = `
apiVersion: saddle/v1
kind: Cluster
metadata:
  name: my-cluster
spec:
  region: us-west-2
  subnetId: subnet-0abc
  securityGroupId: sg-0abc
  ami: ami-0abc
  instanceType: t3.xlarge
  kubernetes:
    distribution: rke2
  ssh:
    keyName: my-key
    privateKeyPath: /tmp/key.pem
    user: ubuntu
`

const multiDocYAML = credentialYAML + "---\n" + clusterYAML

// ── parseBytes ────────────────────────────────────────────────────────────────

func TestParseBytes_SingleCredential(t *testing.T) {
	docs, err := parseBytes([]byte(credentialYAML))
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "Credential", docs[0].kind)
	assert.Equal(t, "prod", docs[0].name)
}

func TestParseBytes_SingleCluster(t *testing.T) {
	docs, err := parseBytes([]byte(clusterYAML))
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "Cluster", docs[0].kind)
	assert.Equal(t, "my-cluster", docs[0].name)
}

func TestParseBytes_MultiDoc(t *testing.T) {
	docs, err := parseBytes([]byte(multiDocYAML))
	require.NoError(t, err)
	assert.Len(t, docs, 2)
	assert.Equal(t, "Credential", docs[0].kind)
	assert.Equal(t, "Cluster", docs[1].kind)
}

func TestParseBytes_MissingKind_Error(t *testing.T) {
	yaml := `
apiVersion: saddle/v1
metadata:
  name: prod
spec:
  provider: aws
`
	_, err := parseBytes([]byte(yaml))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kind")
}

func TestParseBytes_MissingName_Error(t *testing.T) {
	yaml := `
apiVersion: saddle/v1
kind: Credential
metadata:
  name: ""
spec:
  provider: aws
`
	_, err := parseBytes([]byte(yaml))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name")
}

func TestParseBytes_InvalidYAML_Error(t *testing.T) {
	_, err := parseBytes([]byte("{\nbad yaml: ["))
	assert.Error(t, err)
}

func TestParseBytes_EmptyDocument_Skipped(t *testing.T) {
	// Three-dash separator with nothing after should not produce a document.
	yaml := credentialYAML + "\n---\n"
	docs, err := parseBytes([]byte(yaml))
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

// ── parseFile ─────────────────────────────────────────────────────────────────

func TestParseFile_FileNotFound(t *testing.T) {
	_, err := parseFile("/nonexistent/file.yaml")
	assert.Error(t, err)
}

func TestParseFile_ValidFile(t *testing.T) {
	path := writeYAML(t, t.TempDir(), "cred.yaml", credentialYAML)
	docs, err := parseFile(path)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "prod", docs[0].name)
}

func TestParseFile_MultiDocFile(t *testing.T) {
	path := writeYAML(t, t.TempDir(), "all.yaml", multiDocYAML)
	docs, err := parseFile(path)
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}
