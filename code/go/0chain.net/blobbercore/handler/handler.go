package handler

import (
	"context"
	"net/http"
	"runtime/pprof"
	"os"

	"0chain.net/blobbercore/constants"
	"0chain.net/blobbercore/datastore"
	"0chain.net/blobbercore/stats"
	"0chain.net/blobbercore/config"
	"0chain.net/core/common"
	. "0chain.net/core/logging"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

var storageHandler StorageHandler

func GetMetaDataStore() *datastore.Store {
	return datastore.GetStore()
}

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers(r *mux.Router) {
	r.HandleFunc("/v1/file/upload/{allocation}", common.ToJSONResponse(WithConnection(UploadHandler)))
	r.HandleFunc("/v1/file/download/{allocation}", common.ToJSONResponse(WithConnection(DownloadHandler)))
	r.HandleFunc("/v1/file/meta/{allocation}", common.ToJSONResponse(WithReadOnlyConnection(FileMetaHandler)))
	// r.HandleFunc("/v1/file/stats/{allocation}", common.ToJSONResponse(WithReadOnlyConnection(FileStatsHandler)))
	r.HandleFunc("/v1/file/list/{allocation}", common.ToJSONResponse(WithReadOnlyConnection(ListHandler)))
	// r.HandleFunc("/v1/file/objectpath/{allocation}", common.ToJSONResponse(WithReadOnlyConnection(ObjectPathHandler)))
	r.HandleFunc("/v1/file/referencepath/{allocation}", common.ToJSONResponse(WithReadOnlyConnection(ReferencePathHandler)))

	r.HandleFunc("/v1/connection/commit/{allocation}", common.ToJSONResponse(WithConnection(CommitHandler)))
	// r.HandleFunc("/v1/connection/details/{allocation}", common.ToJSONResponse(WithReadOnlyConnection(GetConnectionDetailsHandler)))

	// r.HandleFunc("/v1/readmarker/latest", common.ToJSONResponse(WithReadOnlyConnection(LatestRMHandler)))
	// r.HandleFunc("/v1/challenge/new", common.ToJSONResponse(WithConnection(NewChallengeHandler)))

	// r.HandleFunc("/_metastore", common.ToJSONResponse(WithReadOnlyConnection(MetaStoreHandler)))
	r.HandleFunc("/_debug", common.ToJSONResponse(DumpGoRoutines))
	r.HandleFunc("/_config", common.ToJSONResponse(GetConfig))
	r.HandleFunc("/_stats", stats.StatsHandler)
	// r.HandleFunc("/_retakechallenge", common.ToJSONResponse(RetakeChallenge))

	r.HandleFunc("/allocation", common.ToJSONResponse(WithConnection(AllocationHandler)))
}

func WithReadOnlyConnection(handler common.JSONResponderF) common.JSONResponderF {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		ctx = GetMetaDataStore().CreateTransaction(ctx)
		res, err := handler(ctx, r)
		defer func() {
			GetMetaDataStore().GetTransaction(ctx).Rollback()
		}()
		return res, err
	}
}

func WithConnection(handler common.JSONResponderF) common.JSONResponderF {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		ctx = GetMetaDataStore().CreateTransaction(ctx)
		res, err := handler(ctx, r)
		defer func() {
			if err != nil {
				GetMetaDataStore().GetTransaction(ctx).Rollback()
			}
		}()
		if err != nil {
			return res, err
		}
		err = GetMetaDataStore().GetTransaction(ctx).Commit().Error
		if err != nil {
			return res, common.NewError("commit_error", "Error committing to meta store")
		}
		return res, err
	}
}

func AllocationHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	ctx = context.WithValue(ctx, constants.ALLOCATION_CONTEXT_KEY, vars["allocation"])
	ctx = context.WithValue(ctx, constants.CLIENT_CONTEXT_KEY, r.Header.Get(common.ClientHeader))
	ctx = context.WithValue(ctx, constants.CLIENT_KEY_CONTEXT_KEY, r.Header.Get(common.ClientKeyHeader))

	response, err := storageHandler.GetAllocationDetails(ctx, r)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func FileMetaHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	ctx = context.WithValue(ctx, constants.ALLOCATION_CONTEXT_KEY, vars["allocation"])
	ctx = context.WithValue(ctx, constants.CLIENT_CONTEXT_KEY, r.Header.Get(common.ClientHeader))
	ctx = context.WithValue(ctx, constants.CLIENT_KEY_CONTEXT_KEY, r.Header.Get(common.ClientKeyHeader))

	response, err := storageHandler.GetFileMeta(ctx, r)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/*DownloadHandler is the handler to respond to download requests from clients*/
func DownloadHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	ctx = context.WithValue(ctx, constants.ALLOCATION_CONTEXT_KEY, vars["allocation"])
	ctx = context.WithValue(ctx, constants.CLIENT_CONTEXT_KEY, r.Header.Get(common.ClientHeader))
	ctx = context.WithValue(ctx, constants.CLIENT_KEY_CONTEXT_KEY, r.Header.Get(common.ClientKeyHeader))

	response, err := storageHandler.DownloadFile(ctx, r)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/*ListHandler is the handler to respond to upload requests fro clients*/
func ListHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	ctx = context.WithValue(ctx, constants.CLIENT_CONTEXT_KEY, r.Header.Get(common.ClientHeader))
	ctx = context.WithValue(ctx, constants.CLIENT_KEY_CONTEXT_KEY, r.Header.Get(common.ClientKeyHeader))
	ctx = context.WithValue(ctx, constants.ALLOCATION_CONTEXT_KEY, vars["allocation"])

	response, err := storageHandler.ListEntities(ctx, r)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/*CommitHandler is the handler to respond to upload requests fro clients*/
func CommitHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	ctx = context.WithValue(ctx, constants.ALLOCATION_CONTEXT_KEY, vars["allocation"])
	ctx = context.WithValue(ctx, constants.CLIENT_CONTEXT_KEY, r.Header.Get(common.ClientHeader))
	ctx = context.WithValue(ctx, constants.CLIENT_KEY_CONTEXT_KEY, r.Header.Get(common.ClientKeyHeader))

	response, err := storageHandler.CommitWrite(ctx, r)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func ReferencePathHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	ctx = context.WithValue(ctx, constants.CLIENT_CONTEXT_KEY, r.Header.Get(common.ClientHeader))
	ctx = context.WithValue(ctx, constants.CLIENT_KEY_CONTEXT_KEY, r.Header.Get(common.ClientKeyHeader))
	ctx = context.WithValue(ctx, constants.ALLOCATION_CONTEXT_KEY, vars["allocation"])

	response, err := storageHandler.GetReferencePath(ctx, r)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/*UploadHandler is the handler to respond to upload requests fro clients*/
func UploadHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	ctx = context.WithValue(ctx, constants.ALLOCATION_CONTEXT_KEY, vars["allocation"])
	ctx = context.WithValue(ctx, constants.CLIENT_CONTEXT_KEY, r.Header.Get(common.ClientHeader))
	ctx = context.WithValue(ctx, constants.CLIENT_KEY_CONTEXT_KEY, r.Header.Get(common.ClientKeyHeader))
	Logger.Info("ClientID = ", zap.Any("client_id", r.Header.Get(common.ClientHeader)))
	response, err := storageHandler.WriteFile(ctx, r)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func HandleShutdown(ctx context.Context) {
	go func() {
		select {
		case <-ctx.Done():
			Logger.Info("Shutting down server")
			datastore.GetStore().Close()
		}
	}()
}

func DumpGoRoutines(ctx context.Context, r *http.Request) (interface{}, error) {
	pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	return "success", nil
}

func GetConfig(ctx context.Context, r *http.Request) (interface{}, error) {
	return config.Configuration, nil
}