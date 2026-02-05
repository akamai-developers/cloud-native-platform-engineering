# Build Your Template Library

In deployment steps [#5](../README#5-initialize-your-config) and
[#8](../README#8-add-another-platform) we showed examples of the config file.
You probably observed a default `values` parameter being reused for
both platform definitions. Here is a look at that config file again,
with other params omitted for readability.

```yaml
defaults:
  - &email ruckus@akamai.com
  - &region nl-ams
  - &values values.tpl  # default defined here
...
platform:
  - name: apl-ams
    domain: ams.arch-linux.io
    email: *email 
    region: *region
    repo: github.com/akamai-developers/aplcli
    values: *values  # used here
  ...
  - name: apl-sea
    domain: sea.arch-linux.io
    email: *email 
    region: *region
    repo: github.com/akamai-developers/aplcli
    values: *values  # used again here
    ...
```

This parameter is telling our CLI which template file to use for generating a `values.yaml`
[override](https://helm.sh/docs/chart_template_guide/values_files/)
file for the APL Helm chart. In steps [#6](#6-generate-project-code) and [#8](#8-add-another-platform) we showed a
tree view of the local `.aplcli` application directory. Here is that tree
view again with the Pulumi project code omitted.

```bash
.aplcli
├── config.yaml
└── platforms
    ├── apl-ams
    │   ├── cmd
    │   ├── go.mod
    │   ├── go.sum
    │   └── utils
    ├── apl-sea
    │   ├── cmd
    │   ├── go.mod
    │   ├── go.sum
    │   └── utils
    └── values
        ├── values-example.tpl
        └── values.tpl
```

This default values template was generated when running the `init`
command that created this config directory. The purpose of
templating this versus a hard coding YAML, is so that its reusable
across different platform projects, which are also connected to different Pulumi
ESC environments. As you expand the functionality of this CLI by
adding new commands, refactoring Pulumi code and so on... it's
useful have a single source of truth―one shared directory for all of
your Helm values file templates.

The one created by default provides the minimal overrides necessary for this
implementation of APL, so it's best to leave it be, and instead work on copies
of it when writing new templates. This method also simplifies versioning
templates as you cycle through deployments of APL. For example, we could have
deployed two versions of the Seattle platform (i.e. `prod` and `dev`) with similar but
slightly different templates for each. The application directory could look something
like this:

```bash
.aplcli
├── config.yaml
└── platforms
    ├── apl-ams
    │   ├── cmd
    │   ├── go.mod
    │   ├── go.sum
    │   └── utils
    ├── apl-sea
    │   ├── cmd
    │   ├── go.mod
    │   ├── go.sum
    │   └── utils
    └── values
        ├── values-example.tpl
        ├── values.tpl
        ├── seattle-prod-values.tpl
        └── seattle-dev-values.tpl
```

The corresponding platform definitions in our config file would then be something like:

```yaml
...
platform:
  - name: apl-sea-prod
    domain: sea-prod.arch-linux.io
    email: *email 
    region: us-sea
    values: seattle-prod-values.tpl
  - name: apl-sea-dev
    domain: sea-dev.arch-linux.io
    email: *email 
    region: us-sea
    values: seattle-dev-values.tpl
```

The Helm values for APL gives us a *huge* amount of tunables we can declare to
shape the resulting platforms. For example, both our Seattle and Amsterdam
platforms were provisioned with two teams/tenants (develop and
[admin](https://techdocs.akamai.com/app-platform/docs/platform-teams#admin-team).
If instead we wanted eight different developer teams and six [platform
admins](https://techdocs.akamai.com/app-platform/docs/platform-user-management#user-roles)
in Amsterdam, and four developer teams with 2 platform admins in Seattle―both
with all sorts of messy, overlapping membership and roles―we would define all of
that within these values file templates.

The `values-example.tpl` file demonstrates a more advanced template
that takes advantaged of many other APL features.

> [!IMPORTANT]
> Be aware that APL Helm chart [values
schema](https://github.com/linode/apl-core/blob/main/values-schema.yaml)
often
[changes](https://github.com/linode/apl-core/blob/main/values-changes.yaml)
between releases. Ensure your templates align with your target version
of APL. Not to worry if you forget, the Pulumi code will let you know. :)

```yaml
cluster:
  name: {{ .platformLabel }}
  provider: linode
  domainSuffix: {{ .domain }}
otomi:
  adminPassword: {{ .otomiAdmin }}
  hasExternalDNS: true
dns:
  domainFilters: 
    - {{ .domain }}
  provider:
    linode:
      apiToken: {{ .token }}
apps:
  alertmanager:
    enabled: true
  cert-manager:
    issuer: letsencrypt
    stage: production
    email: admin@{{ .domain }}
  grafana:
    enabled: true
  ingress-nginx-platform:
    _rawValues:
      controller:
        service:
          annotations:
            external-dns.alpha.kubernetes.io/ttl: '30'
            service.beta.kubernetes.io/linode-loadbalancer-tags: '{{ .nodebalancerTag }}'
            service.beta.kubernetes.io/linode-loadbalancer-nodebalancer-id: '{{ .nodebalancerId }}'
  jaeger:
    enabled: true
  harbor:
    enabled: true
  knative:
    enabled: true
  loki:
    enabled: true
    adminPassword: {{ .lokiAdmin }}
  otel:
    enabled: true
  prometheus:
    enabled: true
  rabbitmq:
    enabled: true
  tempo:
    enabled: true 
  trivy:
    enabled: true
  kyverno:
    enabled: true
kms:
  sops:
    provider: age
    age:
      publicKey: '{{ .ageKey }}'
      privateKey: '{{ .agePrivKey }}'
obj:
  provider:
    type: linode
    linode:
      region: {{ .region }}
      accessKeyId: {{ .accessKey }}
      secretAccessKey: {{ .secretKey }}
      buckets:
        {{- range $key, $value := .buckets }}
        {{ $key }}: {{ $value }}
        {{- end }}
platformBackups:
  database:
    gitea:
      enabled: true
      pathSuffix: gitea
      retentionPolicy: 7d
      schedule: 0 0 * * *
    harbor:
      enabled: true
      pathSuffix: harbor
      retentionPolicy: 7d
      schedule: 0 1 * * *
    keycloak:
      enabled: true
      pathSuffix: keycloak
      retentionPolicy: 7d
      schedule: 0 2 * * *
  gitea:
    enabled: true
    retentionPolicy: 7d
    schedule: 0 3 * * *
teamConfig:
  develop:
    settings:
      password: {{ .teamDevelop }}
      id: develop
      selfService:
        teamMembers:
          createServices: true
          editSecurityPolicies: false
          useCloudShell: true
          downloadKubeconfig: true
          downloadDockerLogin: false
      managedMonitoring:
        grafana: true
        alertmanager: true
      networkPolicy:
        egressPublic: true
        ingressPrivate: true
      resourceQuota:
        - name: pods
          value: '50'
        - name: services.loadbalancers
          value: '10'
    builds:
      - name: nodejs-hello-world-v0-0-1
        imageName: nodejs-hello-world
        tag: v0.0.1
        trigger: false
        mode:
          docker:
            repoUrl: https://github.com/linode/apl-nodejs-helloworld
            revision: HEAD
            path: ./Dockerfile
          type: docker
      - name: demo-java-maven-v0-0-1
        imageName: demo-java-maven
        tag: v0.0.1
        trigger: false
        mode:
          buildpacks:
            repoUrl: https://github.com/buildpacks/samples
            revision: HEAD
            path: apps/java-maven
          type: buildpacks
    services:
      - headers:
          response:
            set: []
        id: 78595314-cdaf-4b60-acc2-3b1a7f80fe2b
        ingressClassName: platform
        name: httpbin
        ownHost: true
        port: 80
      - id: a106eb22-8c06-41b6-ab15-97aafb0888b5
        ingressClassName: platform
        name: nginx-deployment
        ownHost: true
        paths: []
        port: 80
    workloads:
      - name: nodejs-helloworld
        url: https://github.com/linode/apl-nodejs-helloworld.git
        path: chart/hello-world
        revision: HEAD
      - name: nginx-deployment
        path: k8s-deployment
        revision: main
        selectedChart: k8s-deployment
        url: https://github.com/linode/apl-charts.git
  admin:
    services: []
    workloads:
      - name: nodejs-helloworld
        url: https://github.com/linode/apl-nodejs-helloworld.git
        path: chart/hello-world
        revision: HEAD
files:
  env/teams/develop/workloadValues/nodejs-helloworld.yaml: |
    values: |
      image:
        repository: otomi/nodejs-helloworld
        tag: v1.2.13
  env/teams/develop/workloadValues/nginx-deployment.yaml: |
    values: |
      fullnameOverride: nginx-deployment
      image:
        repository: nginxinc/nginx-unprivileged
        tag: stable
      containerPorts:
        - containerPort: 8080
          protocol: TCP
          name: http
      resources:
        requests:
          cpu: 200m
          memory: 32Mi
      autoscaling:
        minReplicas: 2
        maxReplicas: 10
  env/teams/admin/workloadValues/nodejs-helloworld.yaml: |
    values: |
      image:
        repository: otomi/nodejs-helloworld
        tag: v1.2.13
users:
  - email: {{ .platformAdminEmail }}
    firstName: Ruckus
    lastName: Voxi
    isPlatformAdmin: true
    isTeamAdmin: false
    teams: []
    initialPassword: {{ randInitPass }}
  - email: rthompson@linode.com
    firstName: Ryan
    lastName: Thompson
    isPlatformAdmin: false
    isTeamAdmin: true
    teams:
      - develop
    initialPassword: {{ randInitPass }}
  - email: jennariley@protonmail.com
    firstName: Jenna
    lastName: Riley
    isPlatformAdmin: false
    isTeamAdmin: false
    teams:
      - develop
    initialPassword: {{ randInitPass }}
  - email: amaeve@cryptq.net
    firstName: Anya
    lastName: Maeve
    isPlatformAdmin: false
    isTeamAdmin: false
    teams:
      - develop
    initialPassword: {{ randInitPass }}
```

____
## Lab Challenge

Create a copy of the `values-example.tpl` located in the `.aplcli/config` directory. Customize and ensure it minimally satisfies the following requirements:

- At least one additional `platform admin` user with a real email address
- Another team (separate from the `develop` team) containing at least two users: one `team admin`, and the other just a standard user
- Limit the total number of services that can be deployed in the new team's namespace
- Register an external repository containing a `Dockerfile` for an image build pipeline
- Update a platform/project [definition](../README#glossary) to use this template instead of the default, and ensure it deploys successfully with APL `v4.12.1`.

Helpful resources for completing this exercise:
- Kubernetes:
  - [Resource Quotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/)
- APL Docs:
  - [Users](https://techdocs.akamai.com/app-platform/docs/platform-user-management)
  - [Team Settings](https://techdocs.akamai.com/app-platform/docs/team-settings)
  - [apl-core values-schema.yaml](https://github.com/linode/apl-core/blob/v4.12.1/values-schema.yaml)


> [!TIP]
> The same user cannot be defined as a `platform admin` and a `team admin`.

### Bonus
In addition to registering an external repository for an image build pipeline, complete the use-case by defining both a workload (deployment) and service for it. Ensure the service is publicly exposed, and reachable from a subdomain in your DNS zone.

> [!TIP]
> Registering a private repository requires extra steps for credential management within APL, in order to give access. An example of doing this _post-installation_ is demonstrated [here](https://github.com/akamai-developers/rag-langgraph-k8s-quickstart?tab=readme-ov-file#12-generate-github-deploy-key). As this is a _pre-installation_ exercise, we recommend using a public repository. If you are up for the challenge however, here are the relevant parts of the [values schema](https://github.com/linode/apl-core/blob/29b154738a5742bc084fa699fa1574d89727ff45/values-schema.yaml#L1359) and [APL documentation](https://techdocs.akamai.com/app-platform/docs/code-repositories) you'll need to pay attention to.
