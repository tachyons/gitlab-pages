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
