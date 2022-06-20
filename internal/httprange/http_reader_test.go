package httprange

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

func TestSeekAndRead(t *testing.T) {
	testServer := newTestServer(t, nil)
	defer testServer.Close()

	resource, err := NewResource(context.Background(), testServer.URL+"/data", testClient)
	require.NoError(t, err)

	tests := map[string]struct {
		readerOffset       int64
		seekOffset         int64
		seekWhence         int
		readSize           int
		expectedContent    string
		expectedSeekErrMsg string
		expectedReadErr    error
	}{
		// io.SeekStart ...
		"read_all_from_seek_start": {
			readSize:        testDataLen,
			seekWhence:      io.SeekStart,
			expectedContent: testData,
			expectedReadErr: io.EOF,
		},
		"read_10_bytes_from_seek_start": {
			readSize:   testDataLen / 3,
			seekWhence: io.SeekStart,
			// "1234567890"
			expectedContent: testData[:testDataLen/3],
			expectedReadErr: nil,
		},
		"read_10_bytes_from_seek_start_with_seek_offset": {
			readSize:   testDataLen / 3,
			seekOffset: int64(testDataLen / 3),
			seekWhence: io.SeekStart,
			// "abcdefghij"
			expectedContent: testData[testDataLen/3 : 2*testDataLen/3],
			expectedReadErr: nil,
		},
		"read_10_bytes_from_seek_offset_until_eof": {
			readSize:   testDataLen / 3,
			seekOffset: int64(2 * testDataLen / 3),
			seekWhence: io.SeekStart,
			// "0987654321"
			expectedContent: testData[2*testDataLen/3:],
			expectedReadErr: io.EOF,
		},
		"read_10_bytes_from_reader_offset_with_seek_offset_to_eof": {
			readSize:     testDataLen / 3,
			readerOffset: int64(testDataLen / 3), // reader offset at "a"
			seekOffset:   int64(testDataLen / 3), // seek offset at "0"
			seekWhence:   io.SeekStart,
			// "0987654321"
			expectedContent: testData[2*testDataLen/3:],
			expectedReadErr: io.EOF,
		},
		"invalid_seek_start_negative_seek_offset": {
			seekOffset:         -1,
			seekWhence:         io.SeekStart,
			expectedSeekErrMsg: "outside of range",
		},
		"invalid_range_seek_at_end": {
			readSize:        testDataLen,
			seekOffset:      int64(testDataLen),
			seekWhence:      io.SeekStart,
			expectedReadErr: vfs.NewReadError(ErrInvalidRange),
		},
		// io.SeekCurrent ...
		"read_all_from_seek_current": {
			readSize:        testDataLen,
			seekWhence:      io.SeekCurrent,
			expectedContent: testData,
			expectedReadErr: io.EOF,
		},
		"read_10_bytes_from_seek_current": {
			readSize:   testDataLen / 3,
			seekWhence: io.SeekCurrent,
			// "1234567890"
			expectedContent: testData[:testDataLen/3],
			expectedReadErr: nil,
		},
		"read_10_bytes_from_seek_current_with_seek_offset": {
			readSize:   testDataLen / 3,
			seekOffset: int64(testDataLen / 3),
			seekWhence: io.SeekCurrent,
			// "abcdefghij"
			expectedContent: testData[testDataLen/3 : 2*testDataLen/3],
			expectedReadErr: nil,
		},
		"read_10_bytes_from_seek_current_with_seek_offset_until_eof": {
			readSize:   testDataLen / 3,
			seekOffset: int64(2 * testDataLen / 3),
			seekWhence: io.SeekCurrent,
			// "0987654321"
			expectedContent: testData[2*testDataLen/3:],
			expectedReadErr: io.EOF,
		},
		"read_10_bytes_from_reader_offset_and_seek_current_with_seek_offset_to_eof": {
			readSize:     testDataLen / 3,
			readerOffset: int64(testDataLen / 3), // reader offset at "a"
			seekOffset:   int64(testDataLen / 3), // seek offset at "0"
			seekWhence:   io.SeekCurrent,
			// "0987654321"
			expectedContent: testData[2*testDataLen/3:],
			expectedReadErr: io.EOF,
		},
		"invalid_seek_current_negative_seek_offset": {
			seekOffset:         -1,
			seekWhence:         io.SeekCurrent,
			expectedSeekErrMsg: "outside of range",
		},
		// io.SeekEnd with negative offsets
		"read_all_from_seek_end": {
			readSize:        testDataLen,
			seekWhence:      io.SeekEnd,
			seekOffset:      -int64(testDataLen),
			expectedContent: testData,
			expectedReadErr: io.EOF,
		},
		"read_10_bytes_from_seek_end": {
			readSize:   testDataLen / 3,
			seekWhence: io.SeekEnd,
			seekOffset: -int64(testDataLen),
			// "1234567890"
			expectedContent: testData[:testDataLen/3],
			expectedReadErr: nil,
		},
		"read_10_bytes_from_seek_end_with_seek_offset": {
			readSize:     testDataLen / 3,
			readerOffset: int64(2 * testDataLen / 3),
			seekOffset:   -int64(testDataLen / 3),
			seekWhence:   io.SeekEnd,
			// "0987654321"
			expectedContent: testData[2*testDataLen/3:],
			expectedReadErr: io.EOF,
		},
		"read_10_bytes_from_seek_end_with_seek_offset_until_eof": {
			readSize:   testDataLen / 3,
			seekOffset: -int64(testDataLen / 3),
			seekWhence: io.SeekEnd,
			// "0987654321"
			expectedContent: testData[2*testDataLen/3:],
			expectedReadErr: io.EOF,
		},
		"read_10_bytes_from_reader_offset_and_seek_end_with_seek_offset_to_eof": {
			readSize:     testDataLen / 3,
			readerOffset: int64(testDataLen / 3),      // reader offset at "a"
			seekOffset:   -int64(2 * testDataLen / 3), // seek offset at "a"
			seekWhence:   io.SeekEnd,
			// "abcdefghij"
			expectedContent: testData[testDataLen/3 : 2*testDataLen/3],
			expectedReadErr: nil,
		},
		"invalid_seek_end_positive_seek_offset": {
			readSize:           testDataLen,
			seekOffset:         1,
			seekWhence:         io.SeekEnd,
			expectedSeekErrMsg: "outside of range",
		},
		"invalid_range_reading_from_end": {
			readSize:        testDataLen / 3,
			seekWhence:      io.SeekEnd,
			expectedReadErr: vfs.NewReadError(ErrInvalidRange),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := NewReader(context.Background(), resource, tt.readerOffset, resource.Size-tt.readerOffset)

			_, err := r.Seek(tt.seekOffset, tt.seekWhence)
			if tt.expectedSeekErrMsg != "" {
				require.EqualError(t, err, tt.expectedSeekErrMsg)
				return
			}
			require.NoError(t, err)

			buf := make([]byte, tt.readSize)
			n, err := r.Read(buf)
			if tt.expectedReadErr != nil {
				require.Equal(t, tt.expectedReadErr, err)
				return
			}

			require.Equal(t, n, tt.readSize)
			require.Equal(t, tt.expectedContent, string(buf))
		})
	}
}

func TestReaderSetResponse(t *testing.T) {
	tests := map[string]struct {
		status          int
		offset          int64
		prevETag        string
		resEtag         string
		expectedErrMsg  string
		expectedIsValid bool
	}{
		"partial_content_success": {
			status:          http.StatusPartialContent,
			expectedIsValid: true,
		},
		"status_ok_success": {
			status:          http.StatusOK,
			expectedIsValid: true,
		},
		"status_ok_previous_response_invalid_offset": {
			status:          http.StatusOK,
			offset:          1,
			expectedErrMsg:  ErrRangeRequestsNotSupported.Error(),
			expectedIsValid: false,
		},
		"status_ok_previous_response_different_etag": {
			status:          http.StatusOK,
			prevETag:        "old",
			resEtag:         "new",
			expectedErrMsg:  ErrRangeRequestsNotSupported.Error(),
			expectedIsValid: false,
		},
		"requested_range_not_satisfiable": {
			status:          http.StatusRequestedRangeNotSatisfiable,
			expectedErrMsg:  ErrRangeRequestsNotSupported.Error(),
			expectedIsValid: false,
		},
		"not_found": {
			status:          http.StatusNotFound,
			expectedErrMsg:  ErrNotFound.Error(),
			expectedIsValid: false,
		},
		"unhandled_status_code": {
			status:          http.StatusInternalServerError,
			expectedErrMsg:  "httprange: read response 500:",
			expectedIsValid: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			resource := &Resource{ETag: tt.prevETag}
			reader := NewReader(context.Background(), resource, tt.offset, 0)
			res := &http.Response{StatusCode: tt.status, Header: map[string][]string{}}
			res.Header.Set("ETag", tt.resEtag)

			err := reader.setResponse(res)

			require.Equal(t, tt.expectedIsValid, resource.Valid())

			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestReaderSeek(t *testing.T) {
	type fields struct {
		Resource   *Resource
		res        *http.Response
		rangeStart int64
		rangeSize  int64
		offset     int64
	}

	tests := map[string]struct {
		fields         fields
		offset         int64
		whence         int
		want           int64
		newOffset      int64
		expectedErrMsg string
	}{
		"invalid_whence": {
			whence:         -1,
			expectedErrMsg: "invalid whence",
		},
		"outside_of_range_invalid_offset": {
			whence:         io.SeekStart,
			offset:         -1,
			fields:         fields{rangeStart: 1},
			expectedErrMsg: "outside of range",
		},
		"outside_of_range_invalid_new_offset": {
			whence:         io.SeekStart,
			offset:         2, // newOffset = 3
			fields:         fields{rangeStart: 1, rangeSize: 1},
			expectedErrMsg: "outside of range",
		},
		"seek_start": {
			whence:    io.SeekStart,
			offset:    1,
			want:      1,
			newOffset: 2,
			fields:    fields{rangeStart: 1, rangeSize: 1},
		},
		"seek_current": {
			whence:    io.SeekCurrent,
			offset:    2,
			want:      1,
			newOffset: 2,
			fields:    fields{rangeStart: 1, rangeSize: 1, offset: 0},
		},
		"seek_end": {
			whence:    io.SeekEnd,
			want:      1,
			newOffset: 2,
			fields:    fields{rangeStart: 1, rangeSize: 1, offset: 0},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := &Reader{
				res:        tt.fields.res,
				rangeStart: tt.fields.rangeStart,
				rangeSize:  tt.fields.rangeSize,
				offset:     tt.fields.offset,
			}

			got, err := r.Seek(tt.offset, tt.whence)
			if tt.expectedErrMsg != "" {
				require.EqualError(t, err, tt.expectedErrMsg)
				return
			}

			require.Equal(t, tt.want, got)
			require.Equal(t, tt.newOffset, r.offset)
		})
	}
}
