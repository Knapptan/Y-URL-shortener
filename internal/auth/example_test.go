package auth_test

import (
	"fmt"

	"github.com/Knapptan/Y-URL-shortener/internal/auth"
)

// ExampleVerifyUserID показывает, как подписать и проверить userID.
func ExampleVerifyUserID() {
	signed := auth.SignUserID("user-42", "my-secret")
	userID, err := auth.VerifyUserID(signed, "my-secret")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(userID)
	// Output: user-42
}
