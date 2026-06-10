package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter/mangapluspb"
	"google.golang.org/protobuf/proto"
)

func protoMarshalResponse(msg proto.Message) []byte {
	success, ok := msg.(*mangapluspb.SuccessResult)
	if !ok {
		resp := &mangapluspb.Response{Result: &mangapluspb.Response_Error{
			Error: &mangapluspb.ErrorResult{Action: mangapluspb.ErrorResult_UNAUTHORIZED},
		}}
		b, _ := proto.Marshal(resp)
		return b
	}
	resp := &mangapluspb.Response{Result: &mangapluspb.Response_Success{Success: success}}
	b, _ := proto.Marshal(resp)
	return b
}

func TestMangaPlusAdapter_FetchLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			regResp := &mangapluspb.SuccessResult{
				RegisterationData: &mangapluspb.RegistrationData{DeviceSecret: "test-secret"},
			}
			w.Write(protoMarshalResponse(regResp))
			return
		}
		titleResp := &mangapluspb.SuccessResult{
			SearchView: &mangapluspb.SearchView{
				AllTitlesGroup: []*mangapluspb.AllTitlesGroup{
					{
						TheTitle: "Popular",
						Titles: []*mangapluspb.Title{
							{TitleId: 1, Name: "One Piece", Author: "Oda", Language: mangapluspb.Language_ENGLISH},
							{TitleId: 2, Name: "Naruto", Author: "Kishimoto", Language: mangapluspb.Language_ENGLISH},
							{TitleId: 3, Name: "Spanish Title", Author: "Author", Language: mangapluspb.Language_SPANISH},
							{TitleId: 1, Name: "One Piece Duplicate", Author: "Oda", Language: mangapluspb.Language_ENGLISH},
						},
					},
				},
			},
		}
		w.Write(protoMarshalResponse(titleResp))
	}))
	defer srv.Close()

	adapter := NewMangaPlusAdapterWithClient(srv.Client(), srv.URL)
	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected 2 English series (deduped), got %d", len(series))
	}
	if series[0].SourceID != "1" || series[0].Title != "One Piece" {
		t.Errorf("unexpected first series: %+v", series[0])
	}
	if series[1].SourceID != "2" || series[1].Title != "Naruto" {
		t.Errorf("unexpected second series: %+v", series[1])
	}
	if series[0].Status != "ONGOING" {
		t.Errorf("expected ONGOING, got %s", series[0].Status)
	}
	if !series[0].IsActive {
		t.Error("expected IsActive=true")
	}
}

func TestMangaPlusAdapter_FetchChapters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			regResp := &mangapluspb.SuccessResult{
				RegisterationData: &mangapluspb.RegistrationData{DeviceSecret: "test-secret"},
			}
			w.Write(protoMarshalResponse(regResp))
			return
		}
		titleResp := &mangapluspb.SuccessResult{
			TitleDetailView: &mangapluspb.TitleDetailView{
				Title: &mangapluspb.Title{TitleId: 100, Name: "Test Manga"},
				ChapterListV2: []*mangapluspb.Chapter{
					{ChapterId: 500, SubTitle: "Chapter 1", StartTimeStamp: 1705276800},
					{ChapterId: 501, SubTitle: "Chapter 2", StartTimeStamp: 1705363200},
					{ChapterId: 500, SubTitle: "Duplicate", StartTimeStamp: 1705276800},
				},
			},
		}
		w.Write(protoMarshalResponse(titleResp))
	}))
	defer srv.Close()

	adapter := NewMangaPlusAdapterWithClient(srv.Client(), srv.URL)
	chapters, err := adapter.FetchChapters(context.Background(), "100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters (deduped), got %d", len(chapters))
	}
	if chapters[0].Number != 500 || chapters[0].Title != "Chapter 1" {
		t.Errorf("unexpected first chapter: %+v", chapters[0])
	}
	if chapters[1].Number != 501 || chapters[1].Title != "Chapter 2" {
		t.Errorf("unexpected second chapter: %+v", chapters[1])
	}
	if chapters[0].IsNew != true {
		t.Error("expected IsNew=true")
	}
}

func TestMangaPlusAdapter_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := &mangapluspb.Response{Result: &mangapluspb.Response_Error{
			Error: &mangapluspb.ErrorResult{Action: mangapluspb.ErrorResult_UNAUTHORIZED},
		}}
		b, _ := proto.Marshal(resp)
		w.Write(b)
	}))
	defer srv.Close()

	adapter := NewMangaPlusAdapterWithClient(srv.Client(), srv.URL)
	_, err := adapter.FetchLatest(context.Background())
	if err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestMangaPlusAdapter_EmptyTitles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			regResp := &mangapluspb.SuccessResult{
				RegisterationData: &mangapluspb.RegistrationData{DeviceSecret: "test-secret"},
			}
			w.Write(protoMarshalResponse(regResp))
			return
		}
		titleResp := &mangapluspb.SuccessResult{
			SearchView: &mangapluspb.SearchView{},
		}
		w.Write(protoMarshalResponse(titleResp))
	}))
	defer srv.Close()

	adapter := NewMangaPlusAdapterWithClient(srv.Client(), srv.URL)
	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 0 {
		t.Errorf("expected 0 series for empty response, got %d", len(series))
	}
}

func TestMangaPlusAdapter_NilSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := &mangapluspb.Response{}
		b, _ := proto.Marshal(resp)
		w.Write(b)
	}))
	defer srv.Close()

	adapter := NewMangaPlusAdapterWithClient(srv.Client(), srv.URL)
	_, err := adapter.FetchLatest(context.Background())
	if err == nil {
		t.Fatal("expected error for nil success")
	}
}

func TestMangaPlusAdapter_InvalidProtobuf(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			regResp := &mangapluspb.SuccessResult{
				RegisterationData: &mangapluspb.RegistrationData{DeviceSecret: "test-secret"},
			}
			w.Write(protoMarshalResponse(regResp))
			return
		}
		w.Write([]byte("invalid-protobuf-data"))
	}))
	defer srv.Close()

	adapter := NewMangaPlusAdapterWithClient(srv.Client(), srv.URL)
	_, err := adapter.FetchLatest(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid protobuf")
	}
}

func TestMangaPlusAdapter_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	adapter := NewMangaPlusAdapterWithClient(srv.Client(), srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := adapter.FetchLatest(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestMangaPlusAdapter_SetSecret(t *testing.T) {
	adapter := NewMangaPlusAdapter()
	adapter.SetSecret("custom-secret")
	if adapter.secret != "custom-secret" {
		t.Errorf("expected secret 'custom-secret', got %q", adapter.secret)
	}
}
