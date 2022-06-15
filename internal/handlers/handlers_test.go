package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/handlers/mock"
)

func TestNotHandleArtifactRequestReturnsFalse(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockArtifact := mock.NewMockArtifact(mockCtrl)
	mockArtifact.EXPECT().
		TryMakeRequest(gomock.Any(), gomock.Any(), "", gomock.Any()).
		Return(false).
		Times(1)

	mockAuth := mock.NewMockAuth(mockCtrl)
	mockAuth.EXPECT().
		GetTokenIfExists(gomock.Any(), gomock.Any()).
		Return("", nil).
		Times(1)

	handlers := New(mockAuth, mockArtifact)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/something")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	require.False(t, handlers.HandleArtifactRequest(result, r))
}

func TestHandleArtifactRequestedReturnsTrue(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockArtifact := mock.NewMockArtifact(mockCtrl)
	mockArtifact.EXPECT().
		TryMakeRequest(gomock.Any(), gomock.Any(), "", gomock.Any()).
		Return(true).
		Times(1)

	mockAuth := mock.NewMockAuth(mockCtrl)
	mockAuth.EXPECT().
		GetTokenIfExists(gomock.Any(), gomock.Any()).
		Return("", nil).
		Times(1)

	handlers := New(mockAuth, mockArtifact)

	result := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/something", nil)

	require.True(t, handlers.HandleArtifactRequest(result, r))
}

func TestNotFoundWithTokenIsNotHandled(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockAuth := mock.NewMockAuth(mockCtrl)
	mockAuth.EXPECT().CheckResponseForInvalidToken(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(false)

	handlers := New(mockAuth, nil)

	w := httptest.NewRecorder()
	reqURL, _ := url.Parse("/")
	r := &http.Request{URL: reqURL}
	response := &http.Response{StatusCode: http.StatusNotFound}
	handled := handlers.checkIfLoginRequiredOrInvalidToken(w, r, "token")(response)

	require.False(t, handled)
}

func TestForbiddenWithTokenIsNotHandled(t *testing.T) {
	cases := map[string]struct {
		StatusCode int
		Token      string
		Handled    bool
	}{
		"403 Forbidden with token": {
			http.StatusForbidden,
			"token",
			false,
		},
		"403 Forbidden with no token": {
			http.StatusForbidden,
			"",
			true,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)

			mockAuth := mock.NewMockAuth(mockCtrl)
			if tc.Token == "" {
				mockAuth.EXPECT().IsAuthSupported().Return(true)
				mockAuth.EXPECT().RequireAuth(gomock.Any(), gomock.Any()).Return(true)
			} else {
				mockAuth.EXPECT().CheckResponseForInvalidToken(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(false)
			}

			handlers := New(mockAuth, nil)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			response := &http.Response{StatusCode: tc.StatusCode}
			handled := handlers.checkIfLoginRequiredOrInvalidToken(w, r, tc.Token)(response)

			require.Equal(t, tc.Handled, handled)
		})
	}
}

func TestNotFoundWithoutTokenIsNotHandledWhenNotAuthSupport(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockAuth := mock.NewMockAuth(mockCtrl)
	mockAuth.EXPECT().IsAuthSupported().Return(false)

	handlers := New(mockAuth, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	response := &http.Response{StatusCode: http.StatusNotFound}
	handled := handlers.checkIfLoginRequiredOrInvalidToken(w, r, "")(response)

	require.False(t, handled)
}
func TestNotFoundWithoutTokenIsHandled(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockAuth := mock.NewMockAuth(mockCtrl)
	mockAuth.EXPECT().IsAuthSupported().Return(true)
	mockAuth.EXPECT().RequireAuth(gomock.Any(), gomock.Any()).Times(1).Return(true)

	handlers := New(mockAuth, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	response := &http.Response{StatusCode: http.StatusNotFound}
	handled := handlers.checkIfLoginRequiredOrInvalidToken(w, r, "")(response)

	require.True(t, handled)
}
func TestInvalidTokenResponseIsHandled(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockAuth := mock.NewMockAuth(mockCtrl)
	mockAuth.EXPECT().CheckResponseForInvalidToken(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(true)

	handlers := New(mockAuth, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	response := &http.Response{StatusCode: http.StatusUnauthorized}
	handled := handlers.checkIfLoginRequiredOrInvalidToken(w, r, "token")(response)

	require.True(t, handled)
}

func TestHandleArtifactRequestButGetTokenFails(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockArtifact := mock.NewMockArtifact(mockCtrl)
	mockArtifact.EXPECT().
		TryMakeRequest(gomock.Any(), gomock.Any(), "", gomock.Any()).
		Times(0)

	mockAuth := mock.NewMockAuth(mockCtrl)
	mockAuth.EXPECT().GetTokenIfExists(gomock.Any(), gomock.Any()).Return("", errors.New("error when retrieving token"))

	handlers := New(mockAuth, mockArtifact)

	result := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/something", nil)

	require.True(t, handlers.HandleArtifactRequest(result, r))
}
