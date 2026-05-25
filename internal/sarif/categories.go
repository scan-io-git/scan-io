package sarif

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

// Category is a security vulnerability taxonomy bucket.
type Category string

const (
	CategoryInjection       Category = "INJECTION"
	CategoryAuth            Category = "AUTH"
	CategoryCrypto          Category = "CRYPTO"
	CategorySecrets         Category = "SECRETS"
	CategoryIDOR            Category = "IDOR"
	CategoryXSS             Category = "XSS"
	CategoryXXE             Category = "XXE"
	CategoryDeserialization Category = "DESERIALIZATION"
	CategorySSRF            Category = "SSRF"
	CategoryPathTraversal   Category = "PATH_TRAVERSAL"
	CategoryCommandInj      Category = "COMMAND_INJECTION"
	CategoryRaceCondition   Category = "RACE_CONDITION"
	CategoryInsecureConfig  Category = "INSECURE_CONFIG"
	CategoryDependency      Category = "DEPENDENCY"
	CategoryMemorySafety    Category = "MEMORY_SAFETY"
	CategoryDOS             Category = "DOS"
	CategoryMassAssignment  Category = "MASS_ASSIGNMENT"
	CategoryLLM             Category = "LLM"
	CategoryOther           Category = "OTHER"
)

var categoryLabels = map[Category]string{
	CategoryInjection:       "Injection",
	CategoryAuth:            "Authentication / authorization",
	CategoryCrypto:          "Cryptographic issue",
	CategorySecrets:         "Hardcoded secret",
	CategoryIDOR:            "IDOR",
	CategoryXSS:             "XSS",
	CategoryXXE:             "XXE",
	CategoryDeserialization: "Insecure deserialization",
	CategorySSRF:            "SSRF",
	CategoryPathTraversal:   "Path traversal",
	CategoryCommandInj:      "Command injection",
	CategoryRaceCondition:   "Race condition",
	CategoryInsecureConfig:  "Insecure configuration",
	CategoryDependency:      "Vulnerable dependency",
	CategoryMemorySafety:    "Memory safety",
	CategoryDOS:             "Denial of service",
	CategoryMassAssignment:  "Mass assignment",
	CategoryLLM:             "LLM / prompt injection",
	CategoryOther:           "Other",
}

// cweToCategory maps CWE IDs to security categories.
// Ported from sec-scan-handler/src/scanner/models/sarif.py:_CWE_TO_CATEGORY.
var cweToCategory = map[int]Category{
	// Injection — SQL/NoSQL/code/EL/LDAP/CRLF/XPath/XQuery
	74: CategoryInjection, 75: CategoryInjection, 89: CategoryInjection,
	90: CategoryInjection, 91: CategoryInjection, 93: CategoryInjection,
	94: CategoryInjection, 95: CategoryInjection, 96: CategoryInjection,
	97: CategoryInjection, 98: CategoryInjection, 99: CategoryInjection,
	113: CategoryInjection, 117: CategoryInjection, 138: CategoryInjection,
	470: CategoryInjection, 471: CategoryInjection, 564: CategoryInjection,
	643: CategoryInjection, 644: CategoryInjection, 652: CategoryInjection,
	917: CategoryInjection, 943: CategoryInjection,
	// XSS
	79: CategoryXSS, 80: CategoryXSS, 83: CategoryXSS, 87: CategoryXSS,
	// XXE
	611: CategoryXXE, 776: CategoryXXE, 827: CategoryXXE,
	// Command injection
	77: CategoryCommandInj, 78: CategoryCommandInj, 88: CategoryCommandInj,
	// Path traversal
	22: CategoryPathTraversal, 23: CategoryPathTraversal, 35: CategoryPathTraversal,
	36: CategoryPathTraversal, 59: CategoryPathTraversal, 73: CategoryPathTraversal,
	// SSRF
	918: CategorySSRF,
	// Deserialization
	502: CategoryDeserialization, 829: CategoryDeserialization, 830: CategoryDeserialization,
	// IDOR
	639: CategoryIDOR,
	// Race condition
	362: CategoryRaceCondition, 364: CategoryRaceCondition, 366: CategoryRaceCondition,
	367: CategoryRaceCondition, 820: CategoryRaceCondition, 833: CategoryRaceCondition,
	// Crypto
	261: CategoryCrypto, 295: CategoryCrypto, 296: CategoryCrypto, 297: CategoryCrypto,
	310: CategoryCrypto, 311: CategoryCrypto, 319: CategoryCrypto, 322: CategoryCrypto,
	323: CategoryCrypto, 324: CategoryCrypto, 325: CategoryCrypto, 326: CategoryCrypto,
	327: CategoryCrypto, 328: CategoryCrypto, 329: CategoryCrypto, 330: CategoryCrypto,
	331: CategoryCrypto, 335: CategoryCrypto, 336: CategoryCrypto, 337: CategoryCrypto,
	338: CategoryCrypto, 340: CategoryCrypto, 347: CategoryCrypto, 757: CategoryCrypto,
	759: CategoryCrypto, 760: CategoryCrypto, 780: CategoryCrypto, 818: CategoryCrypto,
	916: CategoryCrypto,
	// Secrets
	200: CategorySecrets, 201: CategorySecrets, 209: CategorySecrets,
	256: CategorySecrets, 257: CategorySecrets, 259: CategorySecrets,
	312: CategorySecrets, 313: CategorySecrets, 315: CategorySecrets,
	316: CategorySecrets, 318: CategorySecrets, 321: CategorySecrets,
	522: CategorySecrets, 532: CategorySecrets, 538: CategorySecrets,
	540: CategorySecrets, 547: CategorySecrets, 798: CategorySecrets,
	1230: CategorySecrets,
	// Auth
	264: CategoryAuth, 269: CategoryAuth, 275: CategoryAuth, 276: CategoryAuth,
	284: CategoryAuth, 285: CategoryAuth, 287: CategoryAuth, 288: CategoryAuth,
	290: CategoryAuth, 294: CategoryAuth, 302: CategoryAuth, 304: CategoryAuth,
	305: CategoryAuth, 306: CategoryAuth, 307: CategoryAuth, 345: CategoryAuth,
	346: CategoryAuth, 352: CategoryAuth, 384: CategoryAuth, 521: CategoryAuth,
	523: CategoryAuth, 565: CategoryAuth, 601: CategoryAuth, 613: CategoryAuth,
	620: CategoryAuth, 640: CategoryAuth, 732: CategoryAuth, 862: CategoryAuth,
	863: CategoryAuth, 940: CategoryAuth, 1216: CategoryAuth, 1390: CategoryAuth,
	// Insecure config
	2: CategoryInsecureConfig, 11: CategoryInsecureConfig, 13: CategoryInsecureConfig,
	15: CategoryInsecureConfig, 16: CategoryInsecureConfig, 260: CategoryInsecureConfig,
	489: CategoryInsecureConfig, 526: CategoryInsecureConfig, 537: CategoryInsecureConfig,
	541: CategoryInsecureConfig, 548: CategoryInsecureConfig, 552: CategoryInsecureConfig,
	614: CategoryInsecureConfig, 756: CategoryInsecureConfig, 942: CategoryInsecureConfig,
	1004: CategoryInsecureConfig, 1032: CategoryInsecureConfig,
	1174: CategoryInsecureConfig, 1188: CategoryInsecureConfig,
	1275: CategoryInsecureConfig,
	// Dependency
	937: CategoryDependency, 1035: CategoryDependency,
	1104: CategoryDependency, 1395: CategoryDependency,
	// Mass assignment
	915: CategoryMassAssignment,
	// Memory safety
	119: CategoryMemorySafety, 120: CategoryMemorySafety,
	125: CategoryMemorySafety, 416: CategoryMemorySafety,
	476: CategoryMemorySafety, 787: CategoryMemorySafety,
	// DoS
	400: CategoryDOS, 405: CategoryDOS, 770: CategoryDOS, 1333: CategoryDOS,
	// LLM
	913: CategoryLLM, 1426: CategoryLLM, 1427: CategoryLLM,
}

// ruleIDKeywords is an ordered list of (lowercase substring, Category) pairs.
// Used when no CWE tag is present. Ported from sec-scan-handler:_RULEID_CATEGORY_KEYWORDS.
var ruleIDKeywords = []struct {
	kw  string
	cat Category
}{
	{"inject", CategoryInjection},
	{"sql", CategoryInjection},
	{"xss", CategoryXSS},
	{"xxe", CategoryXXE},
	{"ssrf", CategorySSRF},
	{"secret", CategorySecrets},
	{"credential", CategorySecrets},
	{"token", CategorySecrets},
	{"hardcoded", CategorySecrets},
	{"auth", CategoryAuth},
	{"crypto", CategoryCrypto},
	{"idor", CategoryIDOR},
	{"deseializ", CategoryDeserialization},
	{"deserializ", CategoryDeserialization},
	{"path", CategoryPathTraversal},
	{"traversal", CategoryPathTraversal},
	{"command", CategoryCommandInj},
	{"race", CategoryRaceCondition},
	{"config", CategoryInsecureConfig},
	{"depend", CategoryDependency},
}

var (
	cweSemgrepRE = regexp.MustCompile(`^CWE-(\d+):`)
	cweCodeQLRE  = regexp.MustCompile(`(?i)^external/cwe/cwe-(\d+)$`)
)

// resolveCategory derives a Category from rule taxonomy tags, with a rule-ID keyword fallback.
// Returns ("", false) when no signal is present — caller should omit the property.
func resolveCategory(ruleID string, rule *sarif.ReportingDescriptor) (Category, bool) {
	if rule != nil {
		tags, _ := rule.Properties["tags"].([]any)
		for _, t := range tags {
			tag, ok := t.(string)
			if !ok {
				continue
			}
			var m []string
			if m = cweSemgrepRE.FindStringSubmatch(tag); m == nil {
				m = cweCodeQLRE.FindStringSubmatch(tag)
			}
			if m == nil {
				continue
			}
			n, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			if cat, ok := cweToCategory[n]; ok {
				return cat, true
			}
		}
	}

	lower := strings.ToLower(ruleID)
	for _, entry := range ruleIDKeywords {
		if strings.Contains(lower, entry.kw) {
			return entry.cat, true
		}
	}

	return "", false
}
