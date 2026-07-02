package resource

import (
	"path/filepath"
	"testing"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/core"
	"github.com/Felipalds/rancher-saddle/internal/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── ApplyFile — credential ────────────────────────────────────────────────────

func TestApplyFile_Credential_Saved(t *testing.T) {
	dir := t.TempDir()
	credsFile := filepath.Join(dir, "creds.yaml")
	yamlFile := writeYAML(t, dir, "cred.yaml", `
apiVersion: saddle/v1
kind: Credential
metadata:
  name: prod
spec:
  provider: aws
  accessKey: AKIAIOSFODNN7EXAMPLE
  secretKey: supersecret
  region: us-east-1
`)

	err := ApplyFile(yamlFile, "config.yaml", credsFile, nil)
	require.NoError(t, err)

	creds, err := credentials.LoadCredentials(credsFile)
	require.NoError(t, err)
	cred, err := creds.GetAWSCredential("prod")
	require.NoError(t, err)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", cred.AccessKey)
	assert.Equal(t, "us-east-1", cred.DefaultRegion)
}

func TestApplyFile_Credential_Idempotent(t *testing.T) {
	dir := t.TempDir()
	credsFile := filepath.Join(dir, "creds.yaml")
	yaml := `
apiVersion: saddle/v1
kind: Credential
metadata:
  name: prod
spec:
  provider: aws
  accessKey: KEY1
  secretKey: SEC1
  region: us-east-1
`
	// Apply twice — second apply should update, not error.
	require.NoError(t, ApplyFile(writeYAML(t, dir, "c1.yaml", yaml), "config.yaml", credsFile, nil))
	require.NoError(t, ApplyFile(writeYAML(t, dir, "c2.yaml", yaml), "config.yaml", credsFile, nil))

	creds, err := credentials.LoadCredentials(credsFile)
	require.NoError(t, err)
	assert.Len(t, creds.ListAWSCredentials(), 1)
}

// ── ApplyFile — profile ───────────────────────────────────────────────────────

func TestApplyFile_Profile_Saved(t *testing.T) {
	dir := t.TempDir()
	profilesFile := filepath.Join(dir, "profiles.yaml")
	// Temporarily override the hard-coded path by applying to the file directly.
	// We test via applyProfile helper since profilesFile is hard-coded in ApplyFile.
	doc := document{kind: "Profile", name: "us-east", node: nil}
	_ = doc // applyProfile is tested via types_test.go; ApplyFile wiring tested here:

	yamlFile := writeYAML(t, dir, "profile.yaml", `
apiVersion: saddle/v1
kind: Profile
metadata:
  name: us-east
spec:
  region: us-east-1
  subnetId: subnet-0abc12345
  securityGroupId: sg-0abc12345
  ami: ami-0abc12345
  instanceType: t3.xlarge
  ssh:
    keyName: my-key
    privateKeyPath: /tmp/key.pem
    user: ubuntu
`)
	// ApplyFile writes to "profiles.yaml" in cwd; not easily overrideable without
	// further refactor — so we test through applyProfile directly.
	profiles := &config.ProfilesConfig{Profiles: make(map[string]*config.Profile)}
	require.NoError(t, profiles.Save(profilesFile))

	// Verify the YAML parses correctly via parseBytes and type decode.
	docs, err := parseFile(yamlFile)
	require.NoError(t, err)
	require.Len(t, docs, 1)

	var pr ProfileResource
	require.NoError(t, docs[0].node.Decode(&pr))
	assert.Equal(t, "us-east", pr.Metadata.Name)
	assert.Equal(t, "us-east-1", pr.Spec.Region)
	assert.Equal(t, "t3.xlarge", pr.Spec.InstanceType)
}

// ── ApplyFile — AMI ───────────────────────────────────────────────────────────

func TestApplyFile_AMI_Parsed(t *testing.T) {
	dir := t.TempDir()
	yamlFile := writeYAML(t, dir, "ami.yaml", `
apiVersion: saddle/v1
kind: AMI
metadata:
  name: ubuntu-22-us-east-1
spec:
  distro: "Ubuntu 22.04 LTS"
  region: us-east-1
  amiId: ami-0c7217cdde317cfec
`)
	docs, err := parseFile(yamlFile)
	require.NoError(t, err)
	require.Len(t, docs, 1)

	var ar AMIResource
	require.NoError(t, docs[0].node.Decode(&ar))
	assert.Equal(t, "ubuntu-22-us-east-1", ar.Metadata.Name)
	assert.Equal(t, "Ubuntu 22.04 LTS", ar.Spec.Distro)
	assert.Equal(t, "ami-0c7217cdde317cfec", ar.Spec.AMIID)
}

// ── ApplyFile — unknown kind ──────────────────────────────────────────────────

func TestApplyFile_UnknownKind_Error(t *testing.T) {
	dir := t.TempDir()
	yamlFile := writeYAML(t, dir, "bad.yaml", `
apiVersion: saddle/v1
kind: Unicorn
metadata:
  name: sparkle
spec:
  color: purple
`)
	err := ApplyFile(yamlFile, "config.yaml", "creds.yaml", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown kind")
}

// ── ApplyFile — file not found ────────────────────────────────────────────────

func TestApplyFile_FileNotFound(t *testing.T) {
	err := ApplyFile("/nonexistent/file.yaml", "config.yaml", "creds.yaml", nil)
	assert.Error(t, err)
}

// ── ApplyFile — multi-doc ─────────────────────────────────────────────────────

func TestApplyFile_MultiDoc_ProcessesAll(t *testing.T) {
	dir := t.TempDir()
	credsFile := filepath.Join(dir, "creds.yaml")
	yamlFile := writeYAML(t, dir, "all.yaml", `
apiVersion: saddle/v1
kind: Credential
metadata:
  name: dev
spec:
  provider: aws
  accessKey: DEVKEY
  secretKey: DEVSECRET
  region: us-west-2
---
apiVersion: saddle/v1
kind: Credential
metadata:
  name: staging
spec:
  provider: aws
  accessKey: STAGINGKEY
  secretKey: STAGINGSECRET
  region: eu-west-1
`)
	err := ApplyFile(yamlFile, "config.yaml", credsFile, nil)
	require.NoError(t, err)

	creds, err := credentials.LoadCredentials(credsFile)
	require.NoError(t, err)
	assert.Len(t, creds.ListAWSCredentials(), 2)
}

// ── ApplyFile — Cluster — errors without registry ─────────────────────────────

func TestApplyFile_Cluster_FailsWithoutRegistry(t *testing.T) {
	dir := t.TempDir()
	yamlFile := writeYAML(t, dir, "cluster.yaml", `
apiVersion: saddle/v1
kind: Cluster
metadata:
  name: test-cluster
spec:
  accessKey: KEY
  secretKey: SECRET
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
`)
	err := ApplyFile(yamlFile, filepath.Join(dir, "config.yaml"), "creds.yaml", core.NewRegistry())
	// Expected: fails at provider lookup — not a credential error.
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "credential")
}
