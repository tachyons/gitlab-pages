package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
)

func Test_loadMetricsConfig(t *testing.T) {
	defaultMetricsAdress := ":9325"
	defaultDir, defaultMetricsKey, defaultMetricsCertificate := setupHTTPSFixture(t)

	tests := map[string]struct {
		metricsAddress     string
		metricsCertificate string
		metricsKey         string
		expectedError      error
	}{
		"no metrics": {},
		"http metrics": {
			metricsAddress: defaultMetricsAdress,
		},
		"https metrics": {
			metricsAddress:     defaultMetricsAdress,
			metricsCertificate: defaultMetricsCertificate,
			metricsKey:         defaultMetricsKey,
		},
		"https metrics no certificate": {
			metricsAddress: defaultMetricsAdress,
			metricsKey:     defaultMetricsKey,
			expectedError:  errMetricsNoCertificate,
		},
		"https metrics no key": {
			metricsAddress:     defaultMetricsAdress,
			metricsCertificate: defaultMetricsCertificate,
			expectedError:      errMetricsNoKey,
		},
		"https metrics invalid certificate path": {
			metricsAddress:     defaultMetricsAdress,
			metricsCertificate: filepath.Join(defaultDir, "domain.certificate.missing"),
			metricsKey:         defaultMetricsKey,
			expectedError:      os.ErrNotExist,
		},
		"https metrics invalid key path": {
			metricsAddress:     defaultMetricsAdress,
			metricsCertificate: defaultMetricsCertificate,
			metricsKey:         filepath.Join(defaultDir, "domain.key.missing"),
			expectedError:      os.ErrNotExist,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			metricsAddress = &tc.metricsAddress
			metricsCertificate = &tc.metricsCertificate
			metricsKey = &tc.metricsKey
			_, err := loadMetricsConfig()
			require.ErrorIs(t, err, tc.expectedError)
		})
	}
}

func setupHTTPSFixture(t *testing.T) (dir string, key string, cert string) {
	t.Helper()

	tmpDir := t.TempDir()

	keyfile, err := os.CreateTemp(tmpDir, "https-fixture")
	require.NoError(t, err)
	key = keyfile.Name()
	keyfile.Close()

	certfile, err := os.CreateTemp(tmpDir, "https-fixture")
	require.NoError(t, err)
	cert = certfile.Name()
	certfile.Close()

	require.NoError(t, os.WriteFile(key, []byte(fixture.Key), 0644))
	require.NoError(t, os.WriteFile(cert, []byte(fixture.Certificate), 0644))

	return tmpDir, keyfile.Name(), certfile.Name()
}

func TestParseHeaderString(t *testing.T) {
	tests := []struct {
		name          string
		headerStrings []string
		valid         bool
		expectedLen   int
	}{
		{
			name:          "Normal case",
			headerStrings: []string{"X-Test-String: Test"},
			valid:         true,
			expectedLen:   1,
		},
		{
			name:          "Non-tracking header case",
			headerStrings: []string{"Tk: N"},
			valid:         true,
			expectedLen:   1,
		},
		{
			name:          "Content security header case",
			headerStrings: []string{"content-security-policy: default-src 'self'"},
			valid:         true,
			expectedLen:   1,
		},
		{
			name:          "Multiple header strings",
			headerStrings: []string{"content-security-policy: default-src 'self'", "X-Test-String: Test", "My amazing header : Amazing"},
			valid:         true,
			expectedLen:   3,
		},
		{
			name:          "Multiple invalid cases",
			headerStrings: []string{"content-security-policy: default-src 'self'", "test-case"},
			valid:         false,
		},
		{
			name:          "Not valid case",
			headerStrings: []string{"Tk= N"},
			valid:         false,
		},
		{
			name:          "duplicate headers",
			headerStrings: []string{"Tk: N", "Tk: M"},
			valid:         false,
		},
		{
			name:          "Not valid case",
			headerStrings: []string{"X-Test-String Some-Test"},
			valid:         false,
		},
		{
			name:          "Valid and not valid case",
			headerStrings: []string{"content-security-policy: default-src 'self'", "test-case"},
			valid:         false,
		},
		{
			name:          "Multiple headers in single string parsed as one header",
			headerStrings: []string{"content-security-policy: default-src 'self',X-Test-String: Test,My amazing header : Amazing"},
			valid:         true,
			expectedLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHeaderString(tt.headerStrings)
			if tt.valid {
				require.NoError(t, err)
				require.Len(t, got, tt.expectedLen)
				return
			}

			require.Error(t, err)
		})
	}
}
