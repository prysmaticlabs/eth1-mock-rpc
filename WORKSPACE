load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "2697f6bc7c529ee5e6a2d9799870b9ec9eaeb3ee7d70ed50b87a2c2c97e13d9e",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.23.8/rules_go-v0.23.8.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.23.8/rules_go-v0.23.8.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "d8c45ee70ec39a57e7a05e5027c32b1576cc7f16d9dd37135b0eddde45cf1b10",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/v0.20.0/bazel-gazelle-v0.20.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.20.0/bazel-gazelle-v0.20.0.tar.gz",
    ],
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    sha256 = "60821f298a7399450b51b9020394904bbad477c18718d2ad6c789f231e5b8b45",
    strip_prefix = "bazel-tools-a2138311856f55add11cd7009a5abc8d4fd6f163",
    urls = ["https://github.com/atlassian/bazel-tools/archive/a2138311856f55add11cd7009a5abc8d4fd6f163.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "aed1c249d4ec8f703edddf35cbe9dfaca0b5f5ea6e4cd9e83e99f3b0d1136c3d",
    strip_prefix = "rules_docker-0.7.0",
    url = "https://github.com/bazelbuild/rules_docker/archive/v0.7.0.tar.gz",
)

http_archive(
    name = "io_kubernetes_build",
    sha256 = "dd02a62c2a458295f561e280411b04d2efbd97e4954986a401a9a1334cc32cc3",
    strip_prefix = "repo-infra-1b2ddaf3fb8775a5d0f4e28085cf846f915977a8",
    url = "https://github.com/kubernetes/repo-infra/archive/1b2ddaf3fb8775a5d0f4e28085cf846f915977a8.tar.gz",
)

http_archive(
    name = "herumi_bls_eth_go_binary",
    strip_prefix = "bls-eth-go-binary-da18d415993a059052dfed16711f2b3bd03c34b8",
    urls = [
    "https://github.com/herumi/bls-eth-go-binary/archive/da18d415993a059052dfed16711f2b3bd03c34b8.tar.gz",
    ],
    sha256 = "69080ca634f8aaeb0950e19db218811f4bb920a054232e147669ea574ba11ef0",
    build_file = "@com_github_prysmaticlabs_prysm//third_party/herumi:bls_eth_go_binary.BUILD",

)

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

load("@com_github_atlassian_bazel_tools//gometalinter:deps.bzl", "gometalinter_dependencies")

gometalinter_dependencies()

git_repository(
    name = "com_google_protobuf",
    commit = "4cf5bfee9546101d98754d23ff378ff718ba8438",
    remote = "https://github.com/protocolbuffers/protobuf",
    shallow_since = "1558721209 -0700",
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

_go_image_repos()

git_repository(
    name = "graknlabs_bazel_distribution",
    commit = "bd93910450a0f041f5d34a4b97faffcabba21419",
    remote = "https://github.com/graknlabs/bazel-distribution",
    shallow_since = "1563544980 +0300",
)

go_repository(
    name = "com_github_aristanetworks_goarista",
    commit = "728bce664cf5dfb921941b240828f989a2c8f8e3",
    importpath = "github.com/aristanetworks/goarista",
)

go_repository(
    name = "com_github_btcsuite_btcd",
    commit = "306aecffea325e97f513b3ff0cf7895a5310651d",
    importpath = "github.com/btcsuite/btcd",
)

go_repository(
    name = "com_github_go_stack_stack",
    commit = "f66e05c21cd224e01c8a3ee7bc867aa79439e207",  # v1.8.0
    importpath = "github.com/go-stack/stack",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "8dd112bcdc25174059e45e07517d9fc663123347",
    importpath = "golang.org/x/crypto",
)

go_repository(
    name = "com_github_ethereum_go_ethereum",
    commit = "099afb3fd89784f9e3e594b7c2ed11335ca02a9b",
    importpath = "github.com/ethereum/go-ethereum",
    # Note: go-ethereum is not bazel-friendly with regards to cgo. We have a
    # a fork that has resolved these issues by disabling HID/USB support and
    # some manual fixes for c imports in the crypto package. This is forked
    # branch should be updated from time to time with the latest go-ethereum
    # code.
    remote = "https://github.com/prysmaticlabs/bazel-go-ethereum",
    vcs = "git",
)

go_repository(
    name = "com_github_prysmaticlabs_go_ssz",
    commit = "9193cae6b7c3347054706b8466db139b8be90985",
    importpath = "github.com/prysmaticlabs/go-ssz",
)

go_repository(
    name = "com_github_prysmaticlabs_prysm",
    commit = "ec2a100ba992c1a61d66c605c867d0da9777d741",
    importpath = "github.com/prysmaticlabs/prysm",
)

go_repository(
    name = "com_github_phoreproject_bls",
    commit = "da95d4798b09e9f45a29dc53124b2a0b4c1dfc13",
    importpath = "github.com/phoreproject/bls",
)

go_repository(
    name = "com_github_dgraph_io_ristretto",
    commit = "99d1bbbf28e64530eb246be0568fc7709a35ebdd",  # v0.0.1
    importpath = "github.com/dgraph-io/ristretto",
)

go_repository(
    name = "com_github_minio_highwayhash",
    importpath = "github.com/minio/highwayhash",
    commit = "02ca4b43caa3297fbb615700d8800acc7933be98",
)

go_repository(
    name = "in_gopkg_urfave_cli_v2",
    importpath = "gopkg.in/urfave/cli.v2",
    sum = "h1:OvXt/p4cdwNl+mwcWMq/AxaKFkhdxcjx+tx+qf4EOvY=",
    version = "v2.0.0-20190806201727-b62605953717",
)

go_repository(
    name = "com_github_cespare_xxhash",
    commit = "d7df74196a9e781ede915320c11c378c1b2f3a1f",
    importpath = "github.com/cespare/xxhash",
)

go_repository(
    name = "com_github_prysmaticlabs_go_bitfield",
    commit = "ec88cc4d1d143cad98308da54b73d0cdb04254eb",
    importpath = "github.com/prysmaticlabs/go-bitfield",
)

go_repository(
    name = "org_golang_x_net",
    commit = "da137c7871d730100384dbcf36e6f8fa493aef5b",
    importpath = "golang.org/x/net",
)

go_repository(
    name = "org_golang_x_sys",
    commit = "fae7ac547cb717d141c433a2a173315e216b64c4",
    importpath = "golang.org/x/sys",
)

go_repository(
    name = "com_github_pborman_uuid",
    commit = "8b1b92947f46224e3b97bb1a3a5b0382be00d31e",  # v1.2.0
    importpath = "github.com/pborman/uuid",
)

go_repository(
    name = "com_github_google_uuid",
    commit = "0cd6bf5da1e1c83f8b45653022c74f71af0538a4",  # v1.1.1
    importpath = "github.com/google/uuid",
)

go_repository(
    name = "com_github_x_cray_logrus_prefixed_formatter",
    commit = "bb2702d423886830dee131692131d35648c382e2",  # v0.5.2
    importpath = "github.com/x-cray/logrus-prefixed-formatter",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    commit = "e1e72e9de974bd926e5c56f83753fba2df402ce5",  # v1.3.0
    importpath = "github.com/sirupsen/logrus",
)

go_repository(
    name = "com_github_mgutz_ansi",
    commit = "9520e82c474b0a04dd04f8a40959027271bab992",
    importpath = "github.com/mgutz/ansi",
)

go_repository(
    name = "com_github_mattn_go_colorable",
    commit = "8029fb3788e5a4a9c00e415f586a6d033f5d38b3",  # v0.1.2
    importpath = "github.com/mattn/go-colorable",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    commit = "1311e847b0cb909da63b5fecfb5370aa66236465",  # v0.0.8
    importpath = "github.com/mattn/go-isatty",
)

go_repository(
    name = "com_github_rs_cors",
    commit = "9a47f48565a795472d43519dd49aac781f3034fb",  # v1.6.0
    importpath = "github.com/rs/cors",
)

go_repository(
    name = "com_github_deckarep_golang_set",
    commit = "cbaa98ba5575e67703b32b4b19f73c91f3c4159e",  # v1.7.1
    importpath = "github.com/deckarep/golang-set",
)

go_repository(
    name = "com_github_pkg_profile",
    commit = "f6fe06335df110bcf1ed6d4e852b760bfc15beee",
    importpath = "github.com/pkg/profile",
)

load("@com_github_prysmaticlabs_go_ssz//:deps.bzl", "go_ssz_dependencies")

go_ssz_dependencies()

go_repository(
    name = "com_github_minio_sha256_simd",
    commit = "649be62517ba577ad7da146440ebfabeea0fb613",
    importpath = "github.com/minio/sha256-simd",
)

go_repository(
    name = "com_github_pkg_errors",
    commit = "27936f6d90f9c8e1145f11ed52ffffbfdb9e0af7",
    importpath = "github.com/pkg/errors",
)
