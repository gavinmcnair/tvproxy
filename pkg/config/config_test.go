package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEnvDuration_Default(t *testing.T) {
	d := envDuration("TVPROXY_TEST_NONEXISTENT_KEY", 5*time.Minute)
	assert.Equal(t, 5*time.Minute, d)
}

func TestEnvDuration_ValidValue(t *testing.T) {
	os.Setenv("TVPROXY_TEST_DUR", "10m")
	defer os.Unsetenv("TVPROXY_TEST_DUR")

	d := envDuration("TVPROXY_TEST_DUR", 5*time.Minute)
	assert.Equal(t, 10*time.Minute, d)
}

func TestEnvDuration_InvalidValue(t *testing.T) {
	os.Setenv("TVPROXY_TEST_DUR_INVALID", "not-a-duration")
	defer os.Unsetenv("TVPROXY_TEST_DUR_INVALID")

	d := envDuration("TVPROXY_TEST_DUR_INVALID", 5*time.Minute)
	assert.Equal(t, 5*time.Minute, d)
}

func TestEnvDuration_NegativeValue(t *testing.T) {
	os.Setenv("TVPROXY_TEST_DUR_NEG", "-10m")
	defer os.Unsetenv("TVPROXY_TEST_DUR_NEG")

	d := envDuration("TVPROXY_TEST_DUR_NEG", 5*time.Minute)
	assert.Equal(t, 5*time.Minute, d, "negative durations should fall back to default")
}

func TestEnvStr_Default(t *testing.T) {
	s := envStr("TVPROXY_TEST_NONEXISTENT_STR", "fallback")
	assert.Equal(t, "fallback", s)
}

func TestEnvStr_Override(t *testing.T) {
	os.Setenv("TVPROXY_TEST_STR", "custom")
	defer os.Unsetenv("TVPROXY_TEST_STR")

	s := envStr("TVPROXY_TEST_STR", "fallback")
	assert.Equal(t, "custom", s)
}

func TestEnvInt_Default(t *testing.T) {
	i := envInt("TVPROXY_TEST_NONEXISTENT_INT", 42)
	assert.Equal(t, 42, i)
}

func TestEnvInt_Override(t *testing.T) {
	os.Setenv("TVPROXY_TEST_INT", "99")
	defer os.Unsetenv("TVPROXY_TEST_INT")

	i := envInt("TVPROXY_TEST_INT", 42)
	assert.Equal(t, 99, i)
}

func TestEnvBool_Default(t *testing.T) {
	b := envBool("TVPROXY_TEST_NONEXISTENT_BOOL", true)
	assert.True(t, b)
}

func TestEnvBool_Override(t *testing.T) {
	os.Setenv("TVPROXY_TEST_BOOL", "false")
	defer os.Unsetenv("TVPROXY_TEST_BOOL")

	b := envBool("TVPROXY_TEST_BOOL", true)
	assert.False(t, b)
}

func TestLoad_HasRecordStopBuffer(t *testing.T) {
	cfg := Load()
	assert.Equal(t, 5*time.Minute, cfg.RecordStopBuffer)
}

func TestListenAddr(t *testing.T) {
	cfg := &Config{Host: "0.0.0.0", Port: 8080}
	assert.Equal(t, "0.0.0.0:8080", cfg.ListenAddr())
}
