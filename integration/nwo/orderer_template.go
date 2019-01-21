
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司。保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package nwo

const DefaultOrdererTemplate = `---
{{ with $w := . -}}
General:
  LedgerType: file
  ListenAddress: 127.0.0.1
  ListenPort: {{ .OrdererPort Orderer "Listen" }}
  TLS:
    Enabled: true
    PrivateKey: {{ $w.OrdererLocalTLSDir Orderer }}/server.key
    Certificate: {{ $w.OrdererLocalTLSDir Orderer }}/server.crt
    RootCAs:
    -  {{ $w.OrdererLocalTLSDir Orderer }}/ca.crt
    ClientAuthRequired: false
    ClientRootCAs:
  Cluster:
    ClientCertificate: {{ $w.OrdererLocalTLSDir Orderer }}/server.crt
    ClientPrivateKey: {{ $w.OrdererLocalTLSDir Orderer }}/server.key
    DialTimeout: 5s
    RPCTimeout: 7s
    ReplicationBufferSize: 20971520
    ReplicationPullTimeout: 5s
    ReplicationRetryTimeout: 5s
    RootCAs:
    -  {{ $w.OrdererLocalTLSDir Orderer }}/ca.crt
  Keepalive:
    ServerMinInterval: 60s
    ServerInterval: 7200s
    ServerTimeout: 20s
  GenesisMethod: file
  GenesisProfile: {{ .SystemChannel.Profile }}
  GenesisFile: {{ .RootDir }}/{{ .SystemChannel.Name }}_block.pb
  SystemChannel: {{ .SystemChannel.Name }}
  LocalMSPDir: {{ $w.OrdererLocalMSPDir Orderer }}
  LocalMSPID: {{ ($w.Organization Orderer.Organization).MSPID }}
  Profile:
    Enabled: false
    Address: 127.0.0.1:{{ .OrdererPort Orderer "Profile" }}
  BCCSP:
    Default: SW
    SW:
      Hash: SHA2
      Security: 256
      FileKeyStore:
        KeyStore:
  Authentication:
    TimeWindow: 15m
FileLedger:
  Location: {{ .OrdererDir Orderer }}/system
  Prefix: hyperledger-fabric-ordererledger
RAMLedger:
  HistorySize: 1000
{{ if eq .Consensus.Type "kafka" -}}
Kafka:
  Retry:
    ShortInterval: 5s
    ShortTotal: 10m
    LongInterval: 5m
    LongTotal: 12h
    NetworkTimeouts:
      DialTimeout: 10s
      ReadTimeout: 10s
      WriteTimeout: 10s
    Metadata:
      RetryBackoff: 250ms
      RetryMax: 3
    Producer:
      RetryBackoff: 100ms
      RetryMax: 3
    Consumer:
      RetryBackoff: 2s
  Topic:
    ReplicationFactor: 1
  Verbose: false
  TLS:
    Enabled: false
    PrivateKey:
    Certificate:
    RootCAs:
  SASLPlain:
    Enabled: false
    User:
    Password:
  Version:{{ end }}
Debug:
  BroadcastTraceDir:
  DeliverTraceDir:
Consensus:
  WALDir: {{ .OrdererDir Orderer }}/etcdraft/wal
  SnapDir: {{ .OrdererDir Orderer }}/etcdraft/snapshot
Operations:
  ListenAddress: 127.0.0.1:{{ .OrdererPort Orderer "Operations" }}
  TLS:
    Enabled: true
    PrivateKey: {{ $w.OrdererLocalTLSDir Orderer }}/server.key
    Certificate: {{ $w.OrdererLocalTLSDir Orderer }}/server.crt
    RootCAs:
    -  {{ $w.OrdererLocalTLSDir Orderer }}/ca.crt
    ClientAuthRequired: false
    ClientRootCAs:
    -  {{ $w.OrdererLocalTLSDir Orderer }}/ca.crt
Metrics:
  Provider: {{ .MetricsProvider }}
  Statsd:
    Network: udp
    Address: {{ if .StatsdEndpoint }}{{ .StatsdEndpoint }}{{ else }}127.0.0.1:8125{{ end }}
    WriteInterval: 5s
    Prefix: {{ ReplaceAll (ToLower Orderer.ID) "." "_" }}
{{- end }}
`
