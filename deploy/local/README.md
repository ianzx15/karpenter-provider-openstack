# Development Guide: Karpenter Provider for OpenStack

This guide outlines the steps to configure your development environment and run the Karpenter controller (OpenStack provider) locally.

## 1. Prerequisites

Ensure you have the following tools installed and configured:

* **Go** (v1.21+)
* **kubectl** (configured for your development cluster)
* **controller-gen** (to generate CRDs and `deepcopy` code):
    ```bash
    go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
    ```

## 2. Credentials Configuration

The controller requires credentials to communicate with the OpenStack API.

1. **Create `clouds.yaml`** (replace with your actual values):
    ```yaml
    clouds:
      openstack:
        auth:
          auth_url: "https://your-auth-url:5000/v3"
          username: "your-username"
          password: "your-password"
          project_id: "your-project-id"
          user_domain_name: "Default"
        region_name: "RegionOne"
        interface: "public"
        identity_api_version: 3
    ```

2. **Create the Cluster, Namespace, and Secret**:
    ```bash
    kind create cluster --name karpenter-dev
    
    kubectl create namespace karpenter
    
    kubectl create secret generic openstack-cloud-config \
      --from-file=clouds.yaml=clouds.yaml \
      --namespace karpenter
    ```

## 3. Preparing the Cluster (CRDs)

You must install the Custom Resource Definitions (CRDs) that Karpenter uses.

1. **(Optional) Cleanup:** Remove old definitions from previous tests.
    ```bash
    kubectl delete crd nodepools.karpenter.sh nodeclaims.karpenter.sh --ignore-not-found=true
    kubectl delete crd openstacknodeclasses.karpenter.k8s.openstack --ignore-not-found=true 
    ```

2. **Install Karpenter Core CRDs (v1):**
    The `v1` provider requires the `v1` core CRDs.
    ```bash
    kubectl apply -f [https://raw.githubusercontent.com/aws/karpenter-provider-aws/v1.0.0/pkg/apis/crds/karpenter.sh_nodepools.yaml](https://raw.githubusercontent.com/aws/karpenter-provider-aws/v1.0.0/pkg/apis/crds/karpenter.sh_nodepools.yaml)
    kubectl apply -f [https://raw.githubusercontent.com/aws/karpenter-provider-aws/v1.0.0/pkg/apis/crds/karpenter.sh_nodeclaims.yaml](https://raw.githubusercontent.com/aws/karpenter-provider-aws/v1.0.0/pkg/apis/crds/karpenter.sh_nodeclaims.yaml)
    ```

3. **Generate and Install your `OpenStackNodeClass` CRD:**
    Run this from the project root to read your Go code (`pkg/apis/...`) and install the custom definition.
    ```bash
    # (Optional) Generate 'zz_generated.deepcopy.go' if you modified structs
    controller-gen object paths="./..."

    # Generate and apply the CRD YAML
    controller-gen crd paths="./..." output:stdout | kubectl apply -f -
    ```

## 4. Running the Controller Locally

Run the controller directly in your terminal to monitor logs in real-time.

1. **Create a `.env` file with the following content:**
    ```bash
    export OS_AUTH_URL="https://your-url"
    export OS_USERNAME="your-user"
    export OS_PASSWORD="your-password"
    export OS_PROJECT_NAME="your-project-name"
    export OS_DOMAIN_NAME="your-domain"
    export OS_REGION_NAME="your-zone"

    export CLUSTER_NAME="karpenter-openstack-test"
    export LEADER_ELECTION_NAMESPACE=kube-system
    export KUBERNETES_MIN_VERSION="1.19.0"
    export KARPENTER_SERVICE="karpenter"
    export LOG_LEVEL="debug"
    export DISABLE_WEBHOOK="true"
    export RUN_INTEGRATION_TESTS=1 
    ```

2. **Apply to terminal:**
    ```bash
    source .env
    ```

## 5. Testing Provisioning

In a **new terminal**, create the resources that trigger provisioning.

1. **Create `dev-resources.yaml`:**
    * Use **actual UUIDs** from your OpenStack environment for the image and network.

    ```yaml
    apiVersion: karpenter.k8s.openstack/v1openstack
    kind: OpenStackNodeClass
    metadata:
      name: default
    spec:
      imageSelectorTerms:
        - id: "YOUR-GLANCE-IMAGE-UUID" 
      networks:
        - id: "YOUR-NEUTRON-NETWORK-UUID"
      securityGroups:
        - name: "default"
      metadata:
        karpenter.sh/discovery: "karpenter-cluster"
    
    ---
    apiVersion: karpenter.sh/v1
    kind: NodePool
    metadata:
      name: default
    spec:
      template:
        spec:
          requirements:
            - key: kubernetes.io/arch
              operator: In
              values: ["amd64"]
            - key: kubernetes.io/os
              operator: In
              values: ["linux"]
          nodeClassRef:
            group: karpenter.k8s.openstack
            kind: OpenStackNodeClass
            name: default
      limits:
        cpu: 10
      disruption:
        consolidationPolicy: WhenEmptyOrUnderutilized
        consolidateAfter: 1m
    ```

2. **Create `test-pod.yaml` (Inflate Test):**
    Request resources that fit your flavor (e.g., `general.medium`).

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: test-inflate
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: inflate
      template:
        metadata:
          labels:
            app: inflate
        spec:
          terminationGracePeriodSeconds: 0
          containers:
            - name: inflate
              image: public.ecr.aws/eks-distro/kubernetes/pause:3.7
              resources:
                requests:
                  cpu: "1.5" 
                  memory: "1Gi"
    ```

3. **Start the Controller:**
    (Keep this terminal open)
    ```bash
    go run cmd/controller/main.go
    ```

4. **Apply Resources:**
    ```bash
    kubectl apply -f dev-resources.yaml
    kubectl apply -f test-pod.yaml
    ```

5. **Monitor Logs:** You should see `Instance successfully created` in the controller terminal.

## 6. Useful Commands (Debug)

### Force NodePool to "Ready"
If `kubectl get nodepool` shows `READY=False` or `Unknown`, you can force the status update:
```bash
kubectl patch openstacknodeclasses.karpenter.k8s.openstack default --type=merge --subresource=status -p '{"status":{"conditions":[{"type":"Ready","status":"True","lastTransitionTime":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","reason":"Reconciled","message":"Manually set"}]}}'
```
## Environment Cleanup
 ```bash
kubectl delete nodeclaim --all
kubectl delete deployment --all
kubectl delete nodepool --all
 ```
Deleting zombie NodeClaims
Run in terminal:
```
for nc in $(kubectl get nodeclaims -o name); do
  echo "Forcing deletion of ${nc}..."
  kubectl patch ${nc} -p '{"metadata":{"finalizers":null}}' --type=merge
done
```
