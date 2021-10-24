package leetcode

import "testing"

const testAccount = "brents_smurf_account"

func TestFindIfUserCompletedLeetCodeProblemIsTrue(t *testing.T) {
	problemName := "two-sum"
	completed, err := FindIfUserCompletedLeetCodeProblem(testAccount, problemName)

	if err != nil {
		t.Errorf("User: %s, Problem: %s, received error: %v", testAccount, problemName, err)
	}

	if !completed {
		t.Errorf("User: %s, Problem: %s, received false, expected true", testAccount, problemName)
	}
}

func TestFindIfUserCompletedLeetCodeProblemNotAttempted(t *testing.T) {
	problemName := "three-sum"
	completed, err := FindIfUserCompletedLeetCodeProblem(testAccount, problemName)

	if err != nil {
		t.Errorf("User: %s, Problem: %s, received error: %v", testAccount, problemName, err)
	}

	if completed {
		t.Errorf("User: %s, Problem: %s, received true, expected false", testAccount, problemName)
	}
}

func TestFindIfUserCompletedLeetCodeProblemNotAccepted(t *testing.T) {
	problemName := "add-two-numbers"
	completed, err := FindIfUserCompletedLeetCodeProblem(testAccount, problemName)

	if err != nil {
		t.Errorf("User: %s, Problem: %s, received error: %v", testAccount, problemName, err)
	}

	if completed {
		t.Errorf("User: %s, Problem: %s, received true, expected false", testAccount, problemName)
	}
}
