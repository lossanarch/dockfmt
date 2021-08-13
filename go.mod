module github.com/lossanarch/dockfmt

go 1.14

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/genuinetools/pkg v0.0.0-20180717194616-e057fa50f234
	github.com/moby/buildkit v0.5.2-0.20190531034802-7c2b06fae9d2
	github.com/sirupsen/logrus v1.3.0
)

replace (
	github.com/containerd/containerd v1.3.0-0.20190426060238-3a3f0aac8819 => github.com/containerd/containerd v1.3.0-beta.2.0.20190823190603-4a2f61c4f2b4
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc6.0.20190307181833-2b18fe1d885e
	golang.org/x/crypto v0.0.0-20190129210102-0709b304e793 => golang.org/x/crypto v0.0.0-20180904163835-0709b304e793
)
