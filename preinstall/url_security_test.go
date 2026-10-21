package preinstall_test

import (
	"net"
	"testing"

	"git.commsnet.org/commstech/repository-detective/preinstall"
)

func mockPublicDNS() func() {
	orig := preinstall.LookupHostIPsForTests()
	preinstall.SetLookupHostIPsForTests(func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("140.82.121.4")}, nil
	})
	return func() { preinstall.SetLookupHostIPsForTests(orig) }
}

func TestValidateRepoURLRejectsPrivateIPv4Literal(t *testing.T) {
	cases := []string{
		"https://10.0.0.1/o/r",
		"https://192.168.1.1/o/r",
		"https://172.16.0.1/o/r",
		"https://127.0.0.1/o/r",
		"https://169.254.169.254/o/r",
	}
	for _, raw := range cases {
		if _, err := preinstall.ValidateRepoURL(raw, false); err == nil {
			t.Fatalf("expected reject for %s", raw)
		}
	}
}

func TestValidateRepoURLRejectsLoopbackAndLocalhost(t *testing.T) {
	for _, raw := range []string{
		"https://localhost/o/r",
		"https://127.0.0.1/o/r",
		"https://[::1]/o/r",
	} {
		if _, err := preinstall.ValidateRepoURL(raw, false); err == nil {
			t.Fatalf("expected reject for %s", raw)
		}
	}
}

func TestValidateRepoURLRejectsPrivateIPv6(t *testing.T) {
	if _, err := preinstall.ValidateRepoURL("https://[fd12:3456:789a:1::1]/o/r", false); err == nil {
		t.Fatal("expected ULA IPv6 to be rejected")
	}
}

func TestValidateRepoURLRejectsLinkLocal(t *testing.T) {
	if _, err := preinstall.ValidateRepoURL("https://169.254.10.10/o/r", false); err == nil {
		t.Fatal("expected link-local to be rejected")
	}
}

func TestValidateRepoURLRejectsNonHTTPS(t *testing.T) {
	if _, err := preinstall.ValidateRepoURL("http://github.com/o/r", false); err == nil {
		t.Fatal("expected http to be rejected")
	}
}

func TestValidateRepoURLHandlesDNSFailure(t *testing.T) {
	orig := preinstall.LookupHostIPsForTests()
	preinstall.SetLookupHostIPsForTests(func(host string) ([]net.IP, error) {
		return nil, net.ErrClosed
	})
	t.Cleanup(func() { preinstall.SetLookupHostIPsForTests(orig) })

	if _, err := preinstall.ValidateRepoURL("https://does-not-resolve.example/o/r", false); err == nil {
		t.Fatal("expected DNS failure to reject URL")
	}
}

func TestValidateRepoURLRejectsPrivateDNSResult(t *testing.T) {
	orig := preinstall.LookupHostIPsForTests()
	preinstall.SetLookupHostIPsForTests(func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.0.0.5")}, nil
	})
	t.Cleanup(func() { preinstall.SetLookupHostIPsForTests(orig) })

	if _, err := preinstall.ValidateRepoURL("https://public-looking.example/o/r", false); err == nil {
		t.Fatal("expected private resolved IP to be rejected")
	}
}

func TestValidateRepoURLRejectsMalformedURL(t *testing.T) {
	if _, err := preinstall.ValidateRepoURL("https://github.com", false); err == nil {
		t.Fatal("expected missing owner/repo to fail")
	}
}

func TestRevalidateHostBeforeClone(t *testing.T) {
	defer mockPublicDNS()()
	if err := preinstall.RevalidateHost("github.com", false); err != nil {
		t.Fatalf("public host should pass: %v", err)
	}
	if err := preinstall.RevalidateHost("127.0.0.1", false); err == nil {
		t.Fatal("loopback should fail revalidation")
	}
}

func TestGitCloneUsesFixedArgv(t *testing.T) {
	args := preinstall.GitCloneArgsForTests("https://github.com/o/r.git", "/tmp/repo")
	cloneIdx := -1
	for i, a := range args {
		if a == "clone" {
			cloneIdx = i
			break
		}
	}
	if cloneIdx < 0 {
		t.Fatalf("unexpected args (missing clone): %v", args)
	}
	sep := -1
	for i, a := range args {
		if a == "--" {
			sep = i
		}
	}
	if sep < 0 {
		t.Fatal("expected -- separator before clone URL")
	}
	if args[sep+1] != "https://github.com/o/r.git" {
		t.Fatalf("clone URL after --: %v", args)
	}
}
