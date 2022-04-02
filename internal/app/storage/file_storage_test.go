package storage

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestFileStorage_LoadFromBuff(t *testing.T) {
	type fields struct {
		FileAccessMutex *sync.Mutex
		memMap          *MemoryMap
		fileContent     string
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr error
		wantMap map[URL]URL
	}{
		{
			name: "Test case #1",
			fields: fields{
				FileAccessMutex: &sync.Mutex{},
				memMap:          NewMemoryMap(),
				fileContent: `{"ShortURL":"c101c693","LongURL":"https://stackoverflow.com/questions/24886015/how-to-convert-uint32-to-string"}
{"ShortURL":"7d7cbdab","LongURL":"https://ya.ru"}
`,
			},
			wantErr: nil,
			wantMap: map[URL]URL{
				"c101c693": "https://stackoverflow.com/questions/24886015/how-to-convert-uint32-to-string",
				"7d7cbdab": "https://ya.ru",
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &FileStorage{
				FileAccessMutex: tt.fields.FileAccessMutex,
				memMap:          tt.fields.memMap,
				//encoder:         nil,
			}
			buffer := bytes.Buffer{}
			buffer.WriteString(tt.fields.fileContent)
			err := d.LoadFromBuff(&buffer)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantMap, d.memMap.Urls)

		})
	}
}

func TestFileStorage_SaveLongURL(t *testing.T) {
	type context struct {
		FileAccessMutex *sync.Mutex
		memMap          *MemoryMap
		fileContent     string
	}
	type args struct {
		long URL
	}
	tests := []struct {
		name            string
		context         context
		args            args
		want            URL
		wantErr         error
		wantFileContent string
	}{
		{
			name: "Test case #1",
			context: context{
				FileAccessMutex: &sync.Mutex{},
				memMap:          NewMemoryMap(),
			},
			want:    "c101c693",
			args:    args{long: "https://stackoverflow.com/questions/24886015/how-to-convert-uint32-to-string"},
			wantErr: nil,
			wantFileContent: `{"ShortURL":"c101c693","LongURL":"https://stackoverflow.com/questions/24886015/how-to-convert-uint32-to-string"}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := bytes.Buffer{}
			buffer.WriteString(tt.context.fileContent)

			d := &FileStorage{
				FileAccessMutex: tt.context.FileAccessMutex,
				memMap:          tt.context.memMap,
				encoder:         json.NewEncoder(&buffer),
			}
			short, err := d.SaveLongURL(tt.args.long)
			require.Equal(t, tt.wantErr, err)
			assert.Equalf(t, tt.want, short, "SaveLongURL(%v)", tt.args.long)
			assert.Equal(t, tt.wantFileContent, buffer.String())
		})
	}
}
