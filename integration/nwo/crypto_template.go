
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

const DefaultCryptoTemplate = `---
{{ with $w := . -}}
OrdererOrgs:{{ range .OrdererOrgs }}
- Name: {{ .Name }}
  Domain: {{ .Domain }}
  EnableNodeOUs: {{ .EnableNodeOUs }}
  {{- if .CA }}
  CA:{{ if .CA.Hostname }}
    Hostname: {{ .CA.Hostname }}
  {{- end -}}
  {{- end }}
  Specs:{{ range $w.OrderersInOrg .Name }}
  - Hostname: {{ .Name }}
    SANS:
    - localhost
    - 127.0.0.1
    - ::1
  {{- end }}
{{- end }}

PeerOrgs:{{ range .PeerOrgs }}
- Name: {{ .Name }}
  Domain: {{ .Domain }}
  EnableNodeOUs: {{ .EnableNodeOUs }}
  {{- if .CA }}
  CA:{{ if .CA.Hostname }}
    hostname: {{ .CA.Hostname }}
    SANS:
    - localhost
    - 127.0.0.1
    - ::1
  {{- end }}
  {{- end }}
  Users:
    Count: {{ .Users }}
  Specs:{{ range $w.PeersInOrg .Name }}
  - Hostname: {{ .Name }}
    SANS:
    - localhost
    - 127.0.0.1
    - ::1
  {{- end }}
{{- end }}
{{- end }}
`
