package inventory

import "testing"

func TestRedactFields(t *testing.T) {
	secret := "secret"
	public := "public"
	base := []Field{{Key: "token", Label: "Token", Value: &secret, Sensitive: true}, {Key: "os", Label: "OS", Value: &public}}
	for _, tc := range []struct {
		name                    string
		audience                Audience
		length                  int
		secretVisible, redacted bool
	}{
		{"manager", Audience{Sensitive: true}, 2, true, false},
		{"ordinary member", Audience{}, 2, false, true},
		{"guest", Audience{Guest: true}, 1, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := redactFields(base, tc.audience)
			if len(out) != tc.length {
				t.Fatalf("fields %+v", out)
			}
			if tc.audience.Guest {
				if out[0].Key != "os" {
					t.Fatalf("guest fields %+v", out)
				}
				return
			}
			if (out[0].Value != nil) != tc.secretVisible || out[0].Redacted != tc.redacted {
				t.Fatalf("secret %+v", out[0])
			}
		})
	}
	if base[0].Value == nil || base[0].Redacted {
		t.Fatal("source fields were mutated")
	}
}

func TestRedactNoteAreaAndHealth(t *testing.T) {
	title, body, cidr, msg := "secret title", "secret body", "192.0.2.0/24", "internal diagnostic"
	vlan := int32(42)
	for _, tc := range []struct {
		name                                 string
		audience                             Audience
		keepNote, keepSensitive, keepMessage bool
	}{
		{"manager", Audience{Sensitive: true}, true, true, true},
		{"ordinary member", Audience{}, true, false, true},
		{"guest", Audience{Guest: true}, false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			area := Area{CIDR: &cidr, VlanID: &vlan}
			redactArea(&area, tc.audience)
			if (area.CIDR != nil) != tc.keepSensitive || (area.VlanID != nil) != tc.keepSensitive {
				t.Fatalf("area %+v", area)
			}
			note := Note{Title: title, Body: body, IsSensitive: true}
			if redactNote(&note, tc.audience) != tc.keepNote {
				t.Fatalf("note %+v", note)
			}
			if tc.keepNote && ((note.Title != "") != tc.keepSensitive || note.Redacted == tc.keepSensitive) {
				t.Fatalf("note content %+v", note)
			}
			node := Node{Health: Health{Status: "down", Message: &msg}}
			redactNode(&node, tc.audience)
			if node.Health.Status != "down" || (node.Health.Message != nil) != tc.keepMessage {
				t.Fatalf("health %+v", node.Health)
			}
		})
	}
}
