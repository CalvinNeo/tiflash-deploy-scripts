./tests/scripts/it-prepare.sh



# debug test_append
MALLOC_CONF="prof:true" TEST_PARALLEL=4 TEST_BENCH_ROWS=500 RUST_BACKTRACE=1 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints"  -- long_term_append_test::test_append --test-threads 1
TEST_PARALLEL=16 TEST_BENCH_ROWS=3200 RUST_BACKTRACE=1 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints"  -- long_term_append_test::test_append --test-threads 1
TEST_PARALLEL=16 TEST_BENCH_ROWS=3200 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints"  -- long_term_append_test::test_append --test-threads 1

# debug test_many_shards
TEST_PARALLEL=16 TEST_BENCH_ROWS=3200 RUST_BACKTRACE=1 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints"  -- long_term_append_test::test_many_shards --test-threads 1
TEST_UPDATE_DELAY=0 TEST_RUNNER=0 TEST_PARALLEL=32 W=32 TEST_PRESPLIT=4096 TEST_BENCH_ROWS=20000 TEST_CHECK=false RUST_BACKTRACE=1 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- long_term_append_test::test_many_shards --test-threads 1

# release test_append
TEST_PARALLEL=4 TEST_BENCH_ROWS=20000 RUST_BACKTRACE=1 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" --release  -- long_term_append_test::test_append --test-threads 1
TEST_PARALLEL=16 TEST_BENCH_ROWS=20000 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" --release  -- long_term_append_test::test_append --test-threads 1

> "./d4096.log" 2>&1
curl "http://127.0.0.1:8501/debug/pprof/profile?seconds=10&frequency=99"

TEST_PRESPLIT=16000 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- long_term_append_test::test_random_split --test-threads 1
TEST_PRESPLIT=16000 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" --release -- long_term_append_test::test_random_split --test-threads 1

# release test_many_shards
TEST_PARALLEL=32 W=32 TEST_PRESPLIT=4096 TEST_BENCH_ROWS=20000 TEST_CHECK=false RUST_BACKTRACE=1 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" --release -- long_term_append_test::test_many_shards --test-threads 1
TEST_UPDATE_DELAY=0 TEST_RUNNER=0 TEST_PARALLEL=32 W=32 TEST_PRESPLIT=4096 TEST_BENCH_ROWS=20000 TEST_CHECK=false RUST_BACKTRACE=1 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" --release -- long_term_append_test::test_many_shards --test-threads 1

# ===> 在这里
TEST_PRESPLIT=4096 TEST_BENCH_ROWS=80000 TEST_RUNNER=16 TEST_PARALLEL=32 W=32 TEST_UPDATE_DELAY=0 TEST_CHECK=false cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" --release -- long_term_append_test::test_many_shards --test-threads 1

TEST_PRESPLIT=4096 TEST_BENCH_ROWS=80000 TEST_RUNNER=512 TEST_PARALLEL=256 W=32 TEST_UPDATE_DELAY=0 TEST_CHECK=false cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" --release -- long_term_append_test::test_many_shards --test-threads 1


# 改成 benchmark 这样
TEST_LOG_LEVEL=error W=32  cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints"  -- long_term_append_test::test_many_shards_multiple --release --test-threads 1

# test_batch
TEST_DB_BATCH=1000 TEST_Pbulk_splitbulk_splitARALLEL=16 TEST_PRESPLIT=16000 TEST_BENCH_ROWS=3200 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" --release -- long_term_append_test::test_bulk_split --test-threads 1

TEST_DB_BATCH=1000 TEST_PARALLEL=16 TEST_PRESPLIT=16000 TEST_BENCH_ROWS=3200 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- long_term_append_test::test_bulk_split --test-threads 1

cargo test --package tici --lib -- worker::mocks::tests::test_midpoint_bytes

cargo test --package tici --lib -- reader::shard_mirror::tests::test_search
cargo test --package tici --lib -- meta::shard_info::tests::test_splitted_size
cargo test --package tici --lib -- common::shard::tests::test_find_first_overlapped

cargo test --package tici --lib -- common::shard::tests::test_merge



cargo test --package tici_common --lib -- codec::tests::test_previous

TEST_LOG_LEVEL=debug TEST_LOG_FILTER_MODS="tici_shard,tantivy,rustls" 

TEST_LOG_LEVEL=debug TEST_LOG_FILTER_MODS="tici_shard,tantivy,rustls" cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- shard_cache_integration_test::test_shard_split_cache_reallocation --test-threads 1
cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- shard_cache_integration_test --test-threads 1

cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- reader_heartbeat_test --test-threads 1
TEST_SIZE=10000 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- worker_integration_test --test-threads 1
TEST_SIZE=10000 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" --release -- worker_integration_test --test-threads 1
cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- meta_service_test --test-threads 1
cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- meta_service_test:: --test-threads 1
cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- meta_service_test::test_create_shard_t_t --test-threads 1

cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- status_server::test_metrics --test-threads 1
TEST_LOG_FILTER_MODS=tici_shard cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- worker_schedule::test_resche_shards_on_one_wn --test-threads 1
cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- meta_service_test::test_split_strategy --test-threads 1
cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- meta_service_test::test_merge_shard_strategy --test-threads 1
cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- meta_service_test::test_create_shard_f_f --test-threads 1
cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- meta_service_test::test_multiple_indexes --test-threads 1


cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- meta_service_test::compaction::test_compaction --test-threads 1

cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- meta_service_test::test_cmeta_oncurrent_leveled_compactions --test-threads 1
cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- meta_service_test::test_merge_shard_basic --test-threads 1
RUST_BACKTRACE=1 cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- long_term_append_test --test-threads 1

cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- shard_cache_integration_test::test_shard_cache_integration
 --test-threads 1

select * from tici.tici_shard_meta\G

export RUSTFLAGS="--cfg tokio_unstable"
while cargo test -p tici_integration_tests --features="openssl-vendored testexport failpoints" -- meta_service_test::test_concurrent_leveled_compactions --test-threads 1; do sleep 1; done

curl "http://127.0.0.1:8501/debug/pprof/profile"

./jeprof.in --svg "http://127.0.0.1:8501/debug/pprof/heap"

curl "http://127.0.0.1:8501/debug/pprof/heap?jeprof=true&text=svg" > sv.svg