## Профилирование памяти

**Оптимизация:** замена `html/template` на `strings.Builder` в `writeMetricsIndex`.

**Нагрузка:** `GET /` и `POST /updates/` в цикле, сервер в Docker.

**Бенчмарк `GetMetricsIndex`:**
- до: 379 allocs/op, ~19 KB/op
- после: 29 allocs/op, ~11 KB/op

**Сравнение профилей:**
\`\`\`
go tool pprof -top -alloc_space -diff_base=profiles/base.pprof profiles/result.pprof
\`\`\`


File: server
Build ID: 77d987d08cc6e921e4155f89339f12f43b1a83ee
Type: alloc_space
Time: 2026-07-04 18:18:52 MSK
Showing nodes accounting for -285391.88kB, 19.08% of 1495525.64kB total
Dropped 148 nodes (cum <= 7477.63kB)
flat  flat%   sum%        cum   cum%
-138429.42kB  9.26%  9.26% -138429.42kB  9.26%  bytes.growSlice
-89601.37kB  5.99% 15.25% -89601.37kB  5.99%  reflect.unsafe_New
89302.90kB  5.97%  9.28% 89302.90kB  5.97%  internal/bytealg.MakeNoZero
-89090.72kB  5.96% 15.23% -105987.12kB  7.09%  html/template.htmlReplacer
-79873.83kB  5.34% 20.57% -394248.65kB 26.36%  reflect.Value.call
-69121.58kB  4.62% 25.20% -118274.33kB  7.91%  reflect.MakeSlice
-49152.75kB  3.29% 28.48% -49152.75kB  3.29%  reflect.unsafe_NewArray
46635.27kB  3.12% 25.36% 50211.25kB  3.36%  github.com/Radiushina/metrics-receiver.git/internal/repository.(*PostgresRepo).Gauges.func1
37913.44kB  2.54% 22.83% 37913.44kB  2.54%  github.com/Radiushina/metrics-receiver.git/internal/handler.writeSortedGaugeItems
19460.20kB  1.30% 21.53% 19460.20kB  1.30%  net/http.Header.Clone (inline)
-19456.38kB  1.30% 22.83% -19456.38kB  1.30%  internal/strconv.FormatFloat (inline)
-16864.33kB  1.13% 23.96% -362897.96kB 24.27%  github.com/Radiushina/metrics-receiver.git/internal/handler.writeMetricsIndex
14441.38kB  0.97% 22.99% 15521.86kB  1.04%  compress/flate.NewWriter (inline)
13313.19kB  0.89% 22.10% 12801.16kB  0.86%  context.AfterFunc
8193.01kB  0.55% 21.55%  9724.71kB  0.65%  github.com/Radiushina/metrics-receiver.git/internal/repository.(*PostgresRepo).Counters.func1
-8192.27kB  0.55% 22.10% -8192.27kB  0.55%  strings.(*Builder).WriteString (inline)
6657.01kB  0.45% 21.66%  6657.01kB  0.45%  reflect.growslice
6656.87kB  0.45% 21.21% 19458.03kB  1.30%  github.com/jackc/pgx/v5/pgconn/ctxwatch.(*ContextWatcher).Watch
6144.84kB  0.41% 20.80%  6144.84kB  0.41%  net/http.(*Server).newConn (inline)
6144.09kB  0.41% 20.39%  6144.09kB  0.41%  github.com/jackc/pgx/v5/pgtype.scanPlanString.Scan
5120.30kB  0.34% 20.05%  5120.30kB  0.34%  syscall.anyToSockaddr
5120.23kB  0.34% 19.70%  5120.23kB  0.34%  net.sockaddrToTCP
-4101.53kB  0.27% 19.98% -4101.53kB  0.27%  sync.(*Pool).pinSlow
4098kB  0.27% 19.70%  4610.01kB  0.31%  io.ReadAll
-4097.12kB  0.27% 19.98% -4097.12kB  0.27%  github.com/jackc/pgx/v5.(*Conn).getRows
-4096.22kB  0.27% 20.25% -540870.63kB 36.17%  text/template.(*Template).execute
4096.06kB  0.27% 19.98%  4096.06kB  0.27%  github.com/jackc/pgx/v5/pgconn.(*PgConn).makeCommandTag (inline)
-3072.61kB  0.21% 20.18% -3072.61kB  0.21%  context.(*cancelCtx).propagateCancel
-3072.54kB  0.21% 20.39% -3072.54kB  0.21%  go.uber.org/zap/zapcore.init.func2
3072.19kB  0.21% 20.18%  2048.16kB  0.14%  net/http.readTransfer
2640.83kB  0.18% 20.01%  2640.83kB  0.18%  compress/flate.(*dictDecoder).init (inline)
2560.20kB  0.17% 19.84%  2560.20kB  0.17%  github.com/jackc/pgx/v5/pgconn.(*ResultReader).Read
2048.62kB  0.14% 19.70%  2048.62kB  0.14%  net/http.(*Request).WithContext (inline)
2048.52kB  0.14% 19.56%  2048.52kB  0.14%  net/textproto.MIMEHeader.Set (inline)
2048.09kB  0.14% 19.43%  2048.09kB  0.14%  context.WithValue
2048.08kB  0.14% 19.29% -299664.46kB 20.04%  net/http.(*conn).serve
-2048.06kB  0.14% 19.43% -520387.48kB 34.80%  text/template.(*state).walkRange
2048.05kB  0.14% 19.29% 11777.23kB  0.79%  github.com/Radiushina/metrics-receiver.git/internal/handler.unmarshalMetrics
2048.05kB  0.14% 19.15% 20482.37kB  1.37%  github.com/Radiushina/metrics-receiver.git/internal/repository.(*PostgresRepo).updateMetricsBatchOnce
-1543.89kB   0.1% 19.25% -1543.89kB   0.1%  github.com/jackc/pgx/v5/pgxpool.(*connResource).getPoolRows (inline)
1541.04kB   0.1% 19.15%  1541.04kB   0.1%  bufio.NewWriterSize (inline)
-1539.38kB   0.1% 19.25% -1539.38kB   0.1%  github.com/jackc/pgx/v5/pgxpool.(*connResource).getConn (inline)
1536.60kB   0.1% 19.15% -309088.49kB 20.67%  main.NewMux.func1.LoggingMiddleware.1
1536.19kB   0.1% 19.05%  1536.19kB   0.1%  net.newFD (inline)
-1536.15kB   0.1% 19.15% -1536.15kB   0.1%  sync.(*poolChain).pushHead
-1536.07kB   0.1% 19.25% -4096.98kB  0.27%  github.com/jackc/pgx/v5.(*Conn).BeginTx
-1536.04kB   0.1% 19.36% -1536.04kB   0.1%  text/template.(*state).validateType
1536.03kB   0.1% 19.25%  1536.03kB   0.1%  net/textproto.(*Reader).ReadLine (inline)
1536.02kB   0.1% 19.15%  2048.03kB  0.14%  encoding/json.(*decodeState).literalStore
1536.02kB   0.1% 19.05%  1536.02kB   0.1%  net/http.(*connReader).startBackgroundRead
1024.14kB 0.068% 18.98%  9729.18kB  0.65%  encoding/json.Unmarshal
-1024.08kB 0.068% 19.05% -3584.66kB  0.24%  context.withCancel (inline)
-1024.04kB 0.068% 19.12% 55856.58kB  3.73%  github.com/Radiushina/metrics-receiver.git/internal/handler.updateMetricsBatch
1024.03kB 0.068% 19.05% 17921.57kB  1.20%  net/http.(*Server).Serve
-1023.09kB 0.068% 19.12% -2562.24kB  0.17%  encoding/json.Marshal
512.03kB 0.034% 19.08% -1538.72kB   0.1%  encoding/json.newEncodeState
512.02kB 0.034% 19.05%  6656.11kB  0.45%  github.com/jackc/pgx/v5.(*baseRows).Scan
-512.01kB 0.034% 19.08% -2559.70kB  0.17%  github.com/jackc/pgx/v5/pgxpool.(*Pool).BeginTx
0.04kB 2.6e-06% 19.08%  3077.04kB  0.21%  github.com/Radiushina/metrics-receiver.git/internal/handler.logRequestBody (inline)
0     0% 19.08% -138428.98kB  9.26%  bytes.(*Buffer).Write
0     0% 19.08% -138429.42kB  9.26%  bytes.(*Buffer).grow
0     0% 19.08%  2640.83kB  0.18%  compress/flate.NewReader
0     0% 19.08%  2640.83kB  0.18%  compress/gzip.(*Reader).Reset
0     0% 19.08%  2640.83kB  0.18%  compress/gzip.(*Reader).readHeader
0     0% 19.08% 15521.86kB  1.04%  compress/gzip.(*Writer).Write
0     0% 19.08%  2640.83kB  0.18%  compress/gzip.NewReader (inline)
0     0% 19.08% -3584.66kB  0.24%  context.WithCancel
0     0% 19.08%  8705.04kB  0.58%  encoding/json.(*decodeState).array
0     0% 19.08%  2048.03kB  0.14%  encoding/json.(*decodeState).object
0     0% 19.08%  8705.04kB  0.58%  encoding/json.(*decodeState).unmarshal
0     0% 19.08%  8705.04kB  0.58%  encoding/json.(*decodeState).value
0     0% 19.08% -48194.72kB  3.22%  fmt.Fprint
0     0% 19.08%  4610.01kB  0.31%  github.com/Radiushina/metrics-receiver.git/internal/handler.readRequestBody
0     0% 19.08% 16933.97kB  1.13%  github.com/Radiushina/metrics-receiver.git/internal/handler.writeJSON
0     0% 19.08% 19460.20kB  1.30%  github.com/Radiushina/metrics-receiver.git/internal/logger.(*loggingResponseWriter).WriteHeader
0     0% 19.08% 16424.44kB  1.10%  github.com/Radiushina/metrics-receiver.git/internal/middleware.(*gzipResponseWriter).Write
0     0% 19.08% -306528.34kB 20.50%  github.com/Radiushina/metrics-receiver.git/internal/middleware.CompressResponse.func1
0     0% 19.08% -306447.66kB 20.49%  github.com/Radiushina/metrics-receiver.git/internal/middleware.DecompressRequest.func1
0     0% 19.08%  9724.71kB  0.65%  github.com/Radiushina/metrics-receiver.git/internal/repository.(*PostgresRepo).Counters
0     0% 19.08% 50211.25kB  3.36%  github.com/Radiushina/metrics-receiver.git/internal/repository.(*PostgresRepo).Gauges
0     0% 19.08% 20482.37kB  1.37%  github.com/Radiushina/metrics-receiver.git/internal/repository.(*PostgresRepo).UpdateMetricsBatch
0     0% 19.08% 20482.37kB  1.37%  github.com/Radiushina/metrics-receiver.git/internal/repository.(*PostgresRepo).UpdateMetricsBatch.func1
0     0% 19.08% 80418.33kB  5.38%  github.com/Radiushina/metrics-receiver.git/internal/repository.retryPostgres
0     0% 19.08%  9724.71kB  0.65%  github.com/Radiushina/metrics-receiver.git/internal/service.Service.Counters
0     0% 19.08% 50211.25kB  3.36%  github.com/Radiushina/metrics-receiver.git/internal/service.Service.Gauges
0     0% 19.08% 20482.37kB  1.37%  github.com/Radiushina/metrics-receiver.git/internal/service.Service.UpdateMetricsBatch
0     0% 19.08% -304911.09kB 20.39%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
0     0% 19.08% -307041.46kB 20.53%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
0     0% 19.08% 18433.12kB  1.23%  github.com/jackc/pgx/v5.(*Conn).Exec
0     0% 19.08%  2560.11kB  0.17%  github.com/jackc/pgx/v5.(*Conn).Query
0     0% 19.08% 18433.12kB  1.23%  github.com/jackc/pgx/v5.(*Conn).exec
0     0% 19.08% 15873.52kB  1.06%  github.com/jackc/pgx/v5.(*Conn).execPrepared
0     0% 19.08%  2559.60kB  0.17%  github.com/jackc/pgx/v5.(*Conn).execSimpleProtocol
0     0% 19.08%  5120.51kB  0.34%  github.com/jackc/pgx/v5.(*dbTx).Commit
0     0% 19.08% 15873.52kB  1.06%  github.com/jackc/pgx/v5.(*dbTx).Exec
0     0% 19.08%  2047.52kB  0.14%  github.com/jackc/pgx/v5/pgconn.(*PgConn).Exec
0     0% 19.08% 19970.55kB  1.34%  github.com/jackc/pgx/v5/pgconn.(*PgConn).ExecStatement
0     0% 19.08% 17410.52kB  1.16%  github.com/jackc/pgx/v5/pgconn.(*PgConn).execExtendedPrefix
0     0% 19.08%  2560.04kB  0.17%  github.com/jackc/pgx/v5/pgconn.(*PgConn).execExtendedSuffix
0     0% 19.08%  2560.04kB  0.17%  github.com/jackc/pgx/v5/pgconn.(*ResultReader).readUntilRowDescription
0     0% 19.08%  3584.05kB  0.24%  github.com/jackc/pgx/v5/pgconn.(*ResultReader).receiveMessage
0     0% 19.08% -4096.98kB  0.27%  github.com/jackc/pgx/v5/pgxpool.(*Conn).BeginTx
0     0% 19.08%  2560.11kB  0.17%  github.com/jackc/pgx/v5/pgxpool.(*Conn).Query
0     0% 19.08% -1543.89kB   0.1%  github.com/jackc/pgx/v5/pgxpool.(*Conn).getPoolRows (inline)
0     0% 19.08% -1539.38kB   0.1%  github.com/jackc/pgx/v5/pgxpool.(*Pool).Acquire
0     0% 19.08% -2559.70kB  0.17%  github.com/jackc/pgx/v5/pgxpool.(*Pool).Begin (inline)
0     0% 19.08% -2572.45kB  0.17%  github.com/jackc/pgx/v5/pgxpool.(*Pool).Query
0     0% 19.08%  5120.51kB  0.34%  github.com/jackc/pgx/v5/pgxpool.(*Tx).Commit
0     0% 19.08% 15873.52kB  1.06%  github.com/jackc/pgx/v5/pgxpool.(*Tx).Exec
0     0% 19.08%  6656.11kB  0.45%  github.com/jackc/pgx/v5/pgxpool.(*poolRows).Scan
0     0% 19.08% -3585.74kB  0.24%  go.uber.org/zap.(*Logger).check
0     0% 19.08%  2050.38kB  0.14%  go.uber.org/zap/buffer.Pool.Get
0     0% 19.08% -1533.57kB   0.1%  go.uber.org/zap/internal/pool.(*Pool[go.shape.*uint8]).Get (inline)
0     0% 19.08% -4097.29kB  0.27%  go.uber.org/zap/zapcore.(*CheckedEntry).AddCore (inline)
0     0% 19.08%  2565.99kB  0.17%  go.uber.org/zap/zapcore.(*CheckedEntry).Write
0     0% 19.08% -4097.29kB  0.27%  go.uber.org/zap/zapcore.(*ioCore).Check
0     0% 19.08%  2565.99kB  0.17%  go.uber.org/zap/zapcore.(*ioCore).Write
0     0% 19.08%  2565.95kB  0.17%  go.uber.org/zap/zapcore.(*jsonEncoder).EncodeEntry
0     0% 19.08%  1538.72kB   0.1%  go.uber.org/zap/zapcore.(*jsonEncoder).clone
0     0% 19.08% -4097.29kB  0.27%  go.uber.org/zap/zapcore.(*sampler).Check
0     0% 19.08% -4097.29kB  0.27%  go.uber.org/zap/zapcore.getCheckedEntry
0     0% 19.08% -3072.54kB  0.21%  go.uber.org/zap/zapcore.init.New[go.shape.*uint8].func6
0     0% 19.08% -540870.63kB 36.17%  html/template.(*Template).Execute
0     0% 19.08% -105987.12kB  7.09%  html/template.htmlEscaper
0     0% 19.08% 17921.57kB  1.20%  main.(*Server).Run (inline)
0     0% 19.08% -306447.66kB 20.49%  main.NewMux.Recover.func2.1
0     0% 19.08% -362897.96kB 24.27%  main.registerRoutes.(*Handler).GetMetrics.func1
0     0% 19.08% 55856.58kB  3.73%  main.registerRoutes.(*Handler).UpdateMetrics.func6
0     0% 19.08% 17921.57kB  1.20%  main.run.func4
0     0% 19.08%  1536.03kB   0.1%  net.(*TCPAddr).String
0     0% 19.08% 10752.70kB  0.72%  net.(*TCPListener).Accept
0     0% 19.08% 10752.70kB  0.72%  net.(*TCPListener).accept
0     0% 19.08% 11264.70kB  0.75%  net.(*netFD).accept
0     0% 19.08% 17376.91kB  1.16%  net/http.(*Server).ListenAndServe
0     0% 19.08% -2560.27kB  0.17%  net/http.(*conn).close
0     0% 19.08% -2560.27kB  0.17%  net/http.(*conn).finalFlush
0     0% 19.08% -2560.27kB  0.17%  net/http.(*conn).serve.func1
0     0% 19.08% 19460.20kB  1.30%  net/http.(*response).WriteHeader
0     0% 19.08% -307350.25kB 20.55%  net/http.HandlerFunc.ServeHTTP
0     0% 19.08%  2048.52kB  0.14%  net/http.Header.Set (inline)
0     0% 19.08% -2050.75kB  0.14%  net/http.newTextprotoReader
0     0% 19.08% -1536.15kB   0.1%  net/http.putBufioReader
0     0% 19.08%  1536.11kB   0.1%  net/http.putTextprotoReader
0     0% 19.08% -305813.68kB 20.45%  net/http.serverHandler.ServeHTTP
0     0% 19.08% -394248.65kB 26.36%  reflect.Value.Call
0     0% 19.08%  6657.01kB  0.45%  reflect.Value.Grow
0     0% 19.08% -41472.63kB  2.77%  reflect.Value.Set
0     0% 19.08% -41472.63kB  2.77%  reflect.Value.assignTo
0     0% 19.08%  6657.01kB  0.45%  reflect.Value.grow
0     0% 19.08% -41472.63kB  2.77%  reflect.packEface (inline)
0     0% 19.08% -41472.63kB  2.77%  reflect.packEfaceData
0     0% 19.08% -41472.63kB  2.77%  reflect.valueInterface
0     0% 19.08% -19456.38kB  1.30%  strconv.FormatFloat (inline)
0     0% 19.08% 89302.90kB  5.97%  strings.(*Builder).Grow
0     0% 19.08% 89302.90kB  5.97%  strings.(*Builder).grow
0     0% 19.08% -7685.82kB  0.51%  sync.(*Pool).Get
0     0% 19.08% -4101.53kB  0.27%  sync.(*Pool).pin
0     0% 19.08%  3584.22kB  0.24%  syscall.Getsockname
0     0% 19.08% -540870.63kB 36.17%  text/template.(*Template).Execute (inline)
0     0% 19.08% -2560.06kB  0.17%  text/template.(*state).evalArg
0     0% 19.08% -395784.69kB 26.46%  text/template.(*state).evalCall
0     0% 19.08% -395784.69kB 26.46%  text/template.(*state).evalCommand
0     0% 19.08% -395784.69kB 26.46%  text/template.(*state).evalFunction
0     0% 19.08% -395784.69kB 26.46%  text/template.(*state).evalPipeline
0     0% 19.08% -48194.72kB  3.22%  text/template.(*state).printValue
0     0% 19.08% -536774.42kB 35.89%  text/template.(*state).walk
0     0% 19.08% -532677.92kB 35.62%  text/template.(*state).walkIfOrWith
0     0% 19.08% -518339.42kB 34.66%  text/template.(*state).walkRange.func2
0     0% 19.08% -394248.65kB 26.36%  text/template.safeCall