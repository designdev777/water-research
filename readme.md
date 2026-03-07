# Water Research RSS Aggregator (Go)

A Go application that aggregates water technology research from RSS feeds and stores them in SQLite.

## Ubuntu Setup

1. **Install Go**:
   ```bash
   wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
   echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
   echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
   source ~/.bashrc
   go version
