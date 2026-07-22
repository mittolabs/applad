package netguard

import "testing"

func TestCheckAddr_RejectsNonPublic(t *testing.T) {
	for _, addr := range []string{
		"169.254.169.254:80",   // cloud metadata
		"10.0.0.5:5432",        // RFC1918 (postgres)
		"127.0.0.1:8080",       // loopback (api)
		"172.17.0.2:80",        // RFC1918 (docker bridge)
		"192.168.1.1:80",       // RFC1918
		"100.64.0.1:80",        // CGNAT
		"0.0.0.0:80",           // unspecified
		"[::1]:80",             // v6 loopback
		"[fe80::1]:80",         // v6 link-local
		"[fd00::1]:80",         // v6 unique-local
		"[::ffff:10.0.0.1]:80", // v4-mapped private
	} {
		if err := CheckAddr(addr); err == nil {
			t.Errorf("CheckAddr(%q) = nil, want error", addr)
		}
	}
}

func TestCheckAddr_AllowsPublic(t *testing.T) {
	for _, addr := range []string{
		"93.184.216.34:443",
		"8.8.8.8:53",
		"[2606:4700::1111]:443",
	} {
		if err := CheckAddr(addr); err != nil {
			t.Errorf("CheckAddr(%q) = %v, want nil", addr, err)
		}
	}
}

func TestCheckAddr_RejectsUnresolved(t *testing.T) {
	// The Control hook only ever sees resolved addresses; a bare hostname
	// reaching it means something upstream broke, so it must fail closed.
	if err := CheckAddr("metadata.internal:80"); err == nil {
		t.Error("CheckAddr with a hostname = nil, want error")
	}
}
