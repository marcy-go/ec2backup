# 設定
APP_NAME=$(basename `pwd`)
APP_OS="linux windows"
APP_ARCH="386 amd64"

# Go1.4.1をダウンロードする
pushd ~/
curl -s -o go.tar.gz http://https://storage.googleapis.com/golang/go1.4.linux-amd64.tar.gz
tar xzf go.tar.gz
export GOROOT=~/go
export PATH=$GOROOT/bin:$PATH
go version
popd

# goxをインストールする
go get github.com/mitchellh/gox
gox -build-toolchain -os="$APP_OS" -arch="$APP_ARCH"

# gitのコミットからバージョンを採番する
APP_VERSION=$(git log --pretty=format:"%h (%ad)" --date=short -1)
echo APP_VERSION is $APP_VERSION

# 必要なライブラリを集める
go get github.com/awslabs/aws-sdk-go/aws
go get github.com/awslabs/aws-sdk-go/gen/ec2
go get github.com/mitchellh/cli

# クロスコンパイルする
gox -os="$APP_OS" -arch="$APP_ARCH" -output="artifacts/{{.OS}}-{{.Arch}}/$APP_NAME" -ldflags "-X main.version '$APP_VERSION'"
find artifacts
