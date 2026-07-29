package auth

import "strings"

// commonPasswordList is a compact blocklist of the most frequently used and
// most frequently breached passwords. It is deliberately small (a few hundred
// entries) rather than a full breach corpus: it catches the passwords an
// attacker tries first without shipping a multi-megabyte asset. Entries are
// lower-cased; the lookup lower-cases the candidate before matching, so the
// comparison is case-insensitive.
//
// When a project enables the passwordDictionary policy, a password whose exact
// (case-insensitive) value appears here is rejected with password_in_dictionary.
var commonPasswordList = []string{
	"123456", "123456789", "12345678", "1234567", "1234567890", "12345",
	"1234", "111111", "123123", "000000", "654321", "666666", "121212",
	"112233", "789456", "159753", "987654321", "123321", "1234554321",
	"password", "password1", "password123", "passw0rd", "p@ssw0rd", "p@ssword",
	"pass", "passwd", "secret", "letmein", "welcome", "welcome1", "welcome123",
	"admin", "admin123", "administrator", "root", "toor", "guest", "user",
	"login", "changeme", "default", "test", "test123", "testtest", "demo",
	"qwerty", "qwerty123", "qwertyuiop", "qwe123", "1q2w3e4r", "1q2w3e",
	"1qaz2wsx", "zaq12wsx", "asdfgh", "asdfghjkl", "asdf", "asdf1234", "zxcvbn",
	"zxcvbnm", "qazwsx", "qazwsxedc", "abc123", "abcd1234", "abcdef", "abcdefg",
	"a1b2c3", "aaaaaa", "aaaa", "iloveyou", "iloveyou1", "loveyou", "sunshine",
	"princess", "flower", "monkey", "monkey1", "dragon", "shadow", "master",
	"superman", "batman", "trustno1", "hello", "hello123", "helloworld",
	"whatever", "starwars", "computer", "internet", "samsung", "google",
	"michael", "jordan", "jordan23", "harley", "hunter", "ranger", "buster",
	"soccer", "baseball", "football", "football1", "hockey", "basketball",
	"tigger", "charlie", "andrew", "thomas", "robert", "daniel", "joshua",
	"maggie", "jennifer", "jessica", "ashley", "amanda", "nicole", "hannah",
	"michelle", "chelsea", "banana", "cookie", "pepper", "ginger", "chocolate",
	"summer", "winter", "autumn", "spring", "freedom", "whatever1", "ninja",
	"mustang", "corvette", "ferrari", "porsche", "mercedes", "camaro",
	"harleydavidson", "yamaha", "diamond", "silver", "golden", "money",
	"cheese", "orange", "purple", "yellow", "matrix", "access", "secret123",
	"passfree", "nopass", "startrek", "pokemon", "minecraft", "fortnite",
	"snoopy", "garfield", "mickey", "donald", "scooby", "spider", "spiderman",
	"batman1", "welcome2", "office", "windows", "system", "server", "oracle",
	"database", "manager", "service", "backup", "public", "private", "secure",
	"security", "network", "cisco", "linux", "unix", "apple", "iphone",
	"android", "samsung1", "nokia", "blink182", "metallica", "nirvana",
	"greenday", "eminem", "rihanna", "beyonce", "madonna", "prince",
	"elvis", "beatles", "queen", "abcd", "1111", "0000", "2222", "1212",
	"7777", "6969", "4321", "55555", "88888888", "11111111", "123abc",
	"a123456", "123456a", "qwertyu", "asdfjkl", "zxcasd", "poiuyt", "lkjhgf",
	"mnbvcx", "trustme", "iloveu", "babygirl", "babyboy", "sweetie", "honey",
	"angel", "angel1", "killer", "gandalf", "hobbit", "frodo", "legolas",
	"aragorn", "smokey", "bandit", "boomer", "cowboy", "cowboys", "raiders",
	"steelers", "packers", "yankees", "redsox", "chelsea1", "arsenal",
	"liverpool", "barcelona", "realmadrid", "juventus", "manchester", "united",
	"chester", "phoenix", "falcon", "eagle", "eagles", "wolf", "wolves",
	"lion", "tiger", "panther", "jaguar", "cobra", "viper", "hawk", "raven",
	"thunder", "storm", "lightning", "blizzard", "avalanche", "volcano",
	"letmein1", "letmein123", "opensesame", "12341234", "121314", "abc12345",
	"password12", "password1234", "qwerty1", "qwerty12", "1234qwer",
	"zaq1zaq1", "1qazxsw2", "!qaz2wsx", "q1w2e3r4", "q1w2e3", "w3lc0me",
	"adm1n", "r00t", "s3cr3t", "letmein!", "changeit", "temp123", "temporary",
	"newpassword", "newpass", "mypassword", "mypass", "yourpassword",
}

// commonPasswords is the lookup set built from commonPasswordList at init.
var commonPasswords = func() map[string]struct{} {
	m := make(map[string]struct{}, len(commonPasswordList))
	for _, p := range commonPasswordList {
		m[p] = struct{}{}
	}
	return m
}()

// isCommonPassword reports whether the password is an exact (case-insensitive)
// match for a common or breached password in the embedded blocklist.
func isCommonPassword(password string) bool {
	_, ok := commonPasswords[strings.ToLower(strings.TrimSpace(password))]
	return ok
}
