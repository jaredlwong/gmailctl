package cfgtest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	cfg "github.com/mbrt/gmailctl/internal/engine/config/v1alpha3"
	"github.com/mbrt/gmailctl/internal/engine/parser"
)

func TestParseEval(t *testing.T) {
	expr := or(
		and(
			fn(parser.FunctionList, parser.OperationOr,
				"list1.gm.com", "list2@gm.com"),
			fn1(parser.FunctionTo, "me@gmail.com"),
		),
		fn1(parser.FunctionSubject, "Subject"),
		fn(parser.FunctionFrom, parser.OperationOr, "@google.com", "b"),
		fn1(parser.FunctionHas, "Important message"),
		fn1(parser.FunctionHas, "foo@bar.com"),
	)
	eval, err := NewEvaluator(expr)
	if err != nil {
		t.Fatalf("NewEvaluator failed: %v", err)
	}

	tests := []struct {
		name        string
		message     cfg.Message
		expectMatch bool
	}{
		{
			name: "subject",
			message: cfg.Message{
				Subject: "contains subject yes",
			},
			expectMatch: true,
		},
		{
			name: "list with @",
			message: cfg.Message{
				Lists: []string{"list1@gm.com"},
				To:    []string{"me@gmail.com"},
			},
			expectMatch: true,
		},
		{
			name: "from google",
			message: cfg.Message{
				From: "someone@google.com",
			},
			expectMatch: true,
		},
		{
			name: "has from",
			message: cfg.Message{
				From: "foo@bar.com",
			},
			expectMatch: true,
		},
		{
			name: "has body",
			message: cfg.Message{
				Body: "important message",
			},
			expectMatch: true,
		},
		{
			// Gmail's to: uses substring matching, so to:me@gmail.com
			// matches notme@gmail.com (contains "me.gmail.com" after
			// @ -> . normalization).
			name: "list and to substring match",
			message: cfg.Message{
				Lists: []string{"list1@gm.com"},
				To:    []string{"notme@gmail.com"},
			},
			expectMatch: true,
		},
		{
			name: "list but truly different to",
			message: cfg.Message{
				Lists: []string{"list1@gm.com"},
				To:    []string{"someone@otherdomain.com"},
			},
			expectMatch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			match := eval.Match(tc.message)
			assert.Equal(t, tc.expectMatch, match)
		})
	}
}

// TestEmailFieldContainsMatching verifies that email fields (from, to, cc, etc.)
// use substring/contains matching, consistent with Gmail's actual behavior.
// Gmail's from: operator matches any message where the From header contains the
// search term as a substring. For example, from:noreply matches
// noreply@example.com, alerts-noreply@bank.com, etc.
func TestEmailFieldContainsMatching(t *testing.T) {
	tests := []struct {
		name        string
		criteria    parser.CriteriaAST
		message     cfg.Message
		expectMatch bool
	}{
		{
			name:        "from domain matches full email",
			criteria:    fn1(parser.FunctionFrom, "example.com"),
			message:     cfg.Message{From: "user@example.com"},
			expectMatch: true,
		},
		{
			name:        "from username matches full email",
			criteria:    fn1(parser.FunctionFrom, "noreply"),
			message:     cfg.Message{From: "noreply@apply.citizensbank.com"},
			expectMatch: true,
		},
		{
			name:        "from partial domain matches",
			criteria:    fn1(parser.FunctionFrom, "citizensbank.com"),
			message:     cfg.Message{From: "noreply@apply.citizensbank.com"},
			expectMatch: true,
		},
		{
			name:        "from display name matches",
			criteria:    fn1(parser.FunctionFrom, "Tali Berzak"),
			message:     cfg.Message{From: "Tali Berzak <tali@example.com>"},
			expectMatch: true,
		},
		{
			name:        "from no match on unrelated email",
			criteria:    fn1(parser.FunctionFrom, "noreply"),
			message:     cfg.Message{From: "marketing@brand.com"},
			expectMatch: false,
		},
		{
			name:        "from case insensitive",
			criteria:    fn1(parser.FunctionFrom, "BankOfAmerica.com"),
			message:     cfg.Message{From: "alerts@bankofamerica.com"},
			expectMatch: true,
		},
		{
			name:        "to domain matches full email",
			criteria:    fn1(parser.FunctionTo, "gmail.com"),
			message:     cfg.Message{To: []string{"jaredlwong@gmail.com"}},
			expectMatch: true,
		},
		{
			name:        "cc contains match",
			criteria:    fn1(parser.FunctionCc, "team"),
			message:     cfg.Message{Cc: []string{"team-leads@company.com"}},
			expectMatch: true,
		},
		{
			name:        "from suffix with @ prefix still works",
			criteria:    fn1(parser.FunctionFrom, "@google.com"),
			message:     cfg.Message{From: "someone@google.com"},
			expectMatch: true,
		},
		{
			name:        "from suffix with * prefix still works",
			criteria:    fn1(parser.FunctionFrom, "*@google.com"),
			message:     cfg.Message{From: "someone@google.com"},
			expectMatch: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eval, err := NewEvaluator(tc.criteria)
			if err != nil {
				t.Fatalf("NewEvaluator failed: %v", err)
			}
			match := eval.Match(tc.message)
			assert.Equal(t, tc.expectMatch, match)
		})
	}
}
