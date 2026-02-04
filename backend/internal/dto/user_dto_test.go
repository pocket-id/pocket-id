package dto

import (
	"testing"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
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
				Email:       utils.Ptr("test@example.com"),
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			wantErr: "",
		},
		{
			name: "missing username",
			input: UserCreateDto{
				Email:       utils.Ptr("test@example.com"),
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
				Email:    utils.Ptr("test@example.com"),
				LastName: "Doe",
			},
			wantErr: "",
		},
		{
			name: "missing display name",
			input: UserCreateDto{
				Username:  "testuser",
				Email:     utils.Ptr("test@example.com"),
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: "",
		},
		{
			name: "username contains invalid characters",
			input: UserCreateDto{
				Username:    "test/ser",
				Email:       utils.Ptr("test@example.com"),
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
				Email:       utils.Ptr("not-an-email"),
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
				Email:       utils.Ptr("test@example.com"),
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
				Email:       utils.Ptr("test@example.com"),
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
