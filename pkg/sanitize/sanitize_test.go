package sanitize

import (
	"testing"
)

func TestStringAndFilterString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          string
		expectedString string
		expectedFilter string
	}{
		{
			name:           "No Whitespace",
			input:          "Hello",
			expectedString: "Hello",
			expectedFilter: "Hello",
		},
		{
			name:           "Leading and Trailing Spaces",
			input:          "  Hello World  ",
			expectedString: "Hello World",
			expectedFilter: "Hello World",
		},
		{
			name:           "Multiple Words with Tabs",
			input:          "Hello\tWorld",
			expectedString: "Hello\tWorld",
			expectedFilter: "Hello World",
		},
		{
			name:           "Comma Separation",
			input:          "Hello,World",
			expectedString: "Hello,World",
			expectedFilter: "Hello,World",
		},
		{
			name:           "Newlines and Special Characters",
			input:          "Hello\nWorld\r\nTest",
			expectedString: "Hello\nWorld\r\nTest",
			expectedFilter: "Hello,World,Test",
		},
		{
			name:           "Empty String",
			input:          "",
			expectedString: "",
			expectedFilter: "",
		},
		{
			name:           "Whitespace Only",
			input:          "    ",
			expectedString: "",
			expectedFilter: "",
		},
		{
			name:           "Form Feeds and Vertical Tabs",
			input:          "Hello\fWorld\vTest",
			expectedString: "Hello\fWorld\vTest",
			expectedFilter: "HelloWorld,Test",
		},
		{
			name:           "Multiple Special Characters",
			input:          "Test,\nWorld\tForm\fFeed\vVertical",
			expectedString: "Test,\nWorld\tForm\fFeed\vVertical",
			expectedFilter: "Test,World FormFeed,Vertical",
		},
		{
			name:           "Whitespace with Newlines and Tabs",
			input:          " \n\t Test \n World \t ",
			expectedString: "Test \n World",
			expectedFilter: "Test,World",
		},
		{
			name:           "Combination of Special Characters",
			input:          "\t\n\fTest\v,World\n",
			expectedString: "Test\v,World",
			expectedFilter: "Test,World",
		},
		{
			name:           "Special Characters Only",
			input:          "\r\n\t\f\v",
			expectedString: "",
			expectedFilter: "",
		},
		{
			name:           "No Interesting Characters",
			input:          ",\n,\r,\t,\f,\v,",
			expectedString: ",\n,\r,\t,\f,\v,",
			expectedFilter: "",
		},
		{
			name:           "Complex String with Symbols",
			input:          "Hello @ World!",
			expectedString: "Hello @ World!",
			expectedFilter: "Hello @ World!",
		},
		{
			name:           "Whitespace Between Words",
			input:          "Hello    World",
			expectedString: "Hello    World",
			expectedFilter: "Hello World",
		},
		{
			name:           "To the Moon Commas",
			input:          "Hello,,,,,World",
			expectedString: "Hello,,,,,World",
			expectedFilter: "Hello,World",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := String(tt.input); got != tt.expectedString {
				t.Errorf("%d String() = %q, want %q", i, got, tt.expectedString)
			}
			if got := FilterString(tt.input); got != tt.expectedFilter {
				t.Errorf("%d FilterString() = %q, want %q", i, got, tt.expectedFilter)
			}
		})
	}
}
