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
				Email:       "test@example.com",
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			wantErr: "",
		},
		{
			name: "missing username",
			input: UserCreateDto{
				Email:       "test@example.com",
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			wantErr: "Field validation for 'Username' failed on the 'required' tag",
		},
		{
			name: "missing display name",
			input: UserCreateDto{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: "Field validation for 'DisplayName' failed on the 'required' tag",
		},
		{
			name: "username contains invalid characters",
			input: UserCreateDto{
				Username:    "test/ser",
				Email:       "test@example.com",
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			wantErr: "Field validation for 'Username' failed on the 'username' tag",
		},
		{
			name: "invalid email",
			input: UserCreateDto{
				Username:    "testuser",
				Email:       "not-an-email",
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			wantErr: "Field validation for 'Email' failed on the 'email' tag",
		},
		{
			name: "first name too short",
			input: UserCreateDto{
				Username:    "testuser",
				Email:       "test@example.com",
				FirstName:   "",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			wantErr: "Field validation for 'FirstName' failed on the 'required' tag",
		},
		{
			name: "last name too long",
			input: UserCreateDto{
				Username:    "testuser",
				Email:       "test@example.com",
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
