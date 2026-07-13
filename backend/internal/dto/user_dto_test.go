package dto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserCreateDto_Validate(t *testing.T) {
	testCases := []struct {
		name    string
		input   UserCreateDto
		wantErr string
	}{
		{
			name: "valid input",
			input: UserCreateDto{
				Username:    "testuser",
				Email:       new("test@example.com"),
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			wantErr: "",
		},
		{
			name: "missing username",
			input: UserCreateDto{
				Email:       new("test@example.com"),
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			wantErr: "Field validation for 'Username' failed on the 'required' tag",
		},
		{
			name: "missing first name",
			input: UserCreateDto{
				Username: "testuser",
				Email:    new("test@example.com"),
				LastName: "Doe",
			},
			wantErr: "",
		},
		{
			name: "missing display name",
			input: UserCreateDto{
				Username:  "testuser",
				Email:     new("test@example.com"),
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: "",
		},
		{
			name: "username contains invalid characters",
			input: UserCreateDto{
				Username:    "test/ser",
				Email:       new("test@example.com"),
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			wantErr: "Field validation for 'Username' failed on the 'username' tag",
		},
		{
			name: "first name too short",
			input: UserCreateDto{
				Username:    "testuser",
				Email:       new("test@example.com"),
				FirstName:   "",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			wantErr: "",
		},
		{
			name: "last name too long",
			input: UserCreateDto{
				Username:    "testuser",
				Email:       new("test@example.com"),
				FirstName:   "John",
				LastName:    "abcdfghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
				DisplayName: "John Doe",
			},
			wantErr: "Field validation for 'LastName' failed on the 'max' tag",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()

			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestUserCreateDtoResolveEmail(t *testing.T) {
	testCases := []struct {
		name    string
		email   *string
		wantErr bool
	}{
		{name: "empty email", email: nil},
		{name: "exact address", email: new("test@example.com")},
		{name: "invalid address", email: new("not-an-email"), wantErr: true},
		{name: "display name is not an exact address", email: new("Test User <test@example.com>"), wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			errs := (&UserCreateDto{Email: testCase.email}).Resolve(nil)
			if testCase.wantErr {
				require.Len(t, errs, 1)
				require.ErrorContains(t, errs[0], "Field validation for 'Email' failed on the 'email' tag")
				return
			}
			require.Empty(t, errs)
		})
	}
}
