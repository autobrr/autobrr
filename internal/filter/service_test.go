// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package filter

import "testing"

func Test_checkSizeFilter(t *testing.T) {
	type args struct {
		minSize     string
		maxSize     string
		releaseSize uint64
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{name: "test_1", args: args{minSize: "1GB", maxSize: "", releaseSize: 100}, want: false, wantErr: false},
		{name: "test_2", args: args{minSize: "1GB", maxSize: "", releaseSize: 2000000000}, want: true, wantErr: false},
		{name: "test_3", args: args{minSize: "1GB", maxSize: "2.2GB", releaseSize: 2000000000}, want: true, wantErr: false},
		{name: "test_4", args: args{minSize: "1GB", maxSize: "2GIB", releaseSize: 2000000000}, want: true, wantErr: false},
		{name: "test_5", args: args{minSize: "1GB", maxSize: "2GB", releaseSize: 2000000010}, want: false, wantErr: false},
		{name: "test_6", args: args{minSize: "1GB", maxSize: "2GB", releaseSize: 2000000000}, want: false, wantErr: false},
		{name: "test_7", args: args{minSize: "", maxSize: "2GB", releaseSize: 2500000000}, want: false, wantErr: false},
		{name: "test_8", args: args{minSize: "", maxSize: "20GB", releaseSize: 2500000000}, want: true, wantErr: false},
		{name: "test_9", args: args{minSize: "unparseable", maxSize: "20GB", releaseSize: 2500000000}, want: false, wantErr: true},
	}
	for _, tt := range tests {
		s := service{}

		t.Run(tt.name, func(t *testing.T) {
			got, err := s.releaseSizeOkay(tt.args.minSize, tt.args.maxSize, tt.args.releaseSize)
			if err != nil != tt.wantErr {
				t.Errorf("checkSizeFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkSizeFilter() got = %v, want %v", got, tt.want)
			}
		})
	}
}
