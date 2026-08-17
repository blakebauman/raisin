package snsverify

import "testing"

func TestVerifyRequiresSignature(t *testing.T) {
	err := Verify(Envelope{Type: "Notification", Message: "{}"})
	if err == nil {
		t.Fatal("expected error for unsigned envelope")
	}
}

func TestAssertAWSSubscribeURL(t *testing.T) {
	ok := []string{
		"https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription&Token=abc",
		"https://sns.amazonaws.com/?Action=ConfirmSubscription&Token=abc",
	}
	for _, u := range ok {
		if err := assertAWSSubscribeURL(u); err != nil {
			t.Fatalf("%s: %v", u, err)
		}
	}
	bad := []string{
		"http://sns.us-east-1.amazonaws.com/",
		"https://evil.example.com/",
		"https://sns.evil.com.amazonaws.com.attacker.test/",
		"https://not-sns.us-east-1.amazonaws.com/",
	}
	for _, u := range bad {
		if err := assertAWSSubscribeURL(u); err == nil {
			t.Fatalf("expected reject for %s", u)
		}
	}
}
