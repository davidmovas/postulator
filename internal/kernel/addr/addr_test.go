package addr_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/addr"
)

func TestLocal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		host string
		want bool
	}{
		{name: "localhost", host: "localhost", want: true},
		{name: "localhost with a port", host: "localhost:8089", want: true},
		{name: "loopback v4", host: "127.0.0.1", want: true},
		{name: "loopback v4 with a port", host: "127.0.0.1:8089", want: true},
		{name: "loopback v4 outside the first address", host: "127.5.4.3", want: true},
		{name: "loopback v6", host: "[::1]:8089", want: true},
		{name: "loopback v6 without a port", host: "[::1]", want: true},
		{name: "private ten", host: "10.0.0.7", want: true},
		{name: "private one seven two", host: "172.16.4.1", want: true},
		{name: "private one nine two", host: "192.168.1.10:8080", want: true},
		{name: "link local", host: "169.254.10.4", want: true},
		{name: "unique local v6", host: "[fd00::1]:8080", want: true},
		{name: "mdns name", host: "shop.local", want: true},
		{name: "localhost suffix", host: "app.localhost", want: true},
		{name: "test suffix", host: "wordpress.test", want: true},
		{name: "upper case name", host: "SHOP.LOCAL", want: true},
		{name: "public name", host: "example.com", want: false},
		{name: "public name with a port", host: "example.com:8080", want: false},
		{name: "public address", host: "93.184.216.34", want: false},
		{name: "public v6", host: "[2606:2800:220:1::248]:443", want: false},
		{name: "name that merely contains local", host: "localhost.example.com", want: false},
		{name: "empty", host: "", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := addr.Local(tc.host); got != tc.want {
				t.Fatalf("Local(%q) = %t, want %t", tc.host, got, tc.want)
			}
		})
	}
}
