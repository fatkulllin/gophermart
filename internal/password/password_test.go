package password_test

// import (
// 	"strings"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/wanthave/wanthave-api/internal/common/password"
// )

// func TestHash(t *testing.T) {
// 	hash, err := password.Hash("test")

// 	pieces := strings.Split(hash, "$")

// 	assert.Nil(t, err)
// 	assert.Equal(t, pieces[0], "scrypt")
// 	assert.Equal(t, pieces[1], "32768")
// 	assert.Equal(t, pieces[2], "8")
// 	assert.Equal(t, pieces[3], "1")
// 	assert.Len(t, pieces[4], 24)
// 	assert.Len(t, pieces[5], 44)
// }

// func TestCompare(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		hash     string
// 		password string
// 		want     bool
// 		wantErr  string
// 	}{
// 		{
// 			name:     "valid password",
// 			hash:     "scrypt$32768$8$1$LKkHIXDLzEg+veXGMaIz7g==$+mDXBabQpGsWyeRhk9vgZpPXJMyZ5Zg4I/+mBdzkUx0=",
// 			password: "test",
// 			want:     true,
// 			wantErr:  "",
// 		},
// 		{
// 			name:     "invalid password",
// 			hash:     "scrypt$32768$8$1$LKkHIXDLzEg+veXGMaIz7g==$+mDXBabQpGsWyeRhk9vgZpPXJMyZ5Zg4I/+mBdzkUx0=",
// 			password: "TEST",
// 			want:     false,
// 			wantErr:  "",
// 		},
// 		{
// 			name:     "invalid algorithm settings",
// 			hash:     "scrypt$32768$4$1$LKkHIXDLzEg+veXGMaIz7g==$+mDXBabQpGsWyeRhk9vgZpPXJMyZ5Zg4I/+mBdzkUx0=",
// 			password: "test",
// 			want:     false,
// 		},
// 		{
// 			name:     "invalid base64 hash encoding",
// 			hash:     "scrypt$32768$4$1$LKkHIXDLzEg+veXGMaIz7g==$invalid^",
// 			password: "test",
// 			want:     false,
// 			wantErr:  "decoding password hash: illegal base64 data at input byte 7",
// 		},
// 		{
// 			name:     "invalid base64 salt encoding",
// 			hash:     "scrypt$32768$4$1$invalid^$+mDXBabQpGsWyeRhk9vgZpPXJMyZ5Zg4I/+mBdzkUx0=",
// 			password: "test",
// 			want:     false,
// 			wantErr:  "decoding salt: illegal base64 data at input byte 7",
// 		},
// 	}

// 	for _, tc := range tests {
// 		t.Run(tc.name, func(t *testing.T) {
// 			got, gotErr := password.Compare(tc.hash, tc.password)

// 			assert.Equal(t, got, tc.want)
// 			if tc.wantErr != "" {
// 				assert.EqualError(t, gotErr, tc.wantErr)
// 			} else {
// 				assert.Nil(t, gotErr)
// 			}
// 		})
// 	}
// }
