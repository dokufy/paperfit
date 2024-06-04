package middleware

import (
	"github.com/atoncetti/combiner/internal/cacher"
	"github.com/danielgtaylor/huma/v2"
)

func GetPdfCacheResponseMiddleware(api huma.API, cacher *cacher.Cacher) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		next(ctx)
		// var b bytes.Buffer

		// raw := ctx.BodyReader()

		// r, _ := io.ReadAll(ctx.BodyReader())

		// err := json.Unmarshal(r, )

		// fs:///home/antonio/projects/private/dokufy/combiner/test/cache
		// storage, err := storage.New("minio://testing/pdf?credential=hmac:0t1nP2rMFBTmz2kbpeOq:yU0JLU4wVrOjDsqCO8E8Mlw1tR0AYUBVtdkuWSZJ&endpoint=http:localhost:9000", logger.New())
		// storage, err := storage.New("fs:///home/antonio/projects/private/dokufy/combiner/test/cache", logger.New())
		// if err != nil {
		// 	huma.WriteErr(api, ctx, http.StatusInternalServerError,
		// 		"Some friendly message", fmt.Errorf("error detail"),
		// 	)

		// 	return
		// }

		// cache := cache.New(storage)

		// log.Printf("%s", str)
		// sha := base64.URLEncoding.EncodeToString(crypto.SHA1.New().Sum([]byte(str)))
		// log.Printf("%s", sha)
		// value, err := cache.Get(ctx.Context(), sha)
	}
}
