#!/bin/bash
set -euo pipefail

NAMESPACE="manga-cdc"
CONTEXT="${1:-minikube}"

echo "Deploying manga-cdc to Kubernetes context: $CONTEXT"
kubectl config use-context "$CONTEXT"

kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/initdb-configmap.yaml

if [ ! -f k8s/secret.yaml ]; then
  echo "Creating dev secret..."
  kubectl create secret generic manga-cdc-secrets \
    --namespace="$NAMESPACE" \
    --from-literal=postgres-password=mangacdc \
    --dry-run=client -o yaml > k8s/secret.yaml
fi
kubectl apply -f k8s/secret.yaml

kubectl apply -f k8s/postgres-service.yaml
kubectl apply -f k8s/postgres-statefulset.yaml
kubectl apply -f k8s/redpanda-service.yaml
kubectl apply -f k8s/redpanda-statefulset.yaml

echo "Waiting for stateful services..."
kubectl wait --for=condition=ready pod -l app=postgres -n "$NAMESPACE" --timeout=120s
kubectl wait --for=condition=ready pod -l app=redpanda -n "$NAMESPACE" --timeout=120s

kubectl apply -f k8s/connect-deployment.yaml
kubectl apply -f k8s/connect-service.yaml
kubectl apply -f k8s/scraper-deployment.yaml
kubectl apply -f k8s/scraper-service.yaml
kubectl apply -f k8s/notification-deployment.yaml
kubectl apply -f k8s/notification-service.yaml
kubectl apply -f k8s/kafkaui-deployment.yaml
kubectl apply -f k8s/kafkaui-service.yaml
kubectl apply -f k8s/prometheus-deployment.yaml
kubectl apply -f k8s/prometheus-service.yaml
kubectl apply -f k8s/grafana-deployment.yaml
kubectl apply -f k8s/grafana-service.yaml

echo "Waiting for stateless services..."
kubectl wait --for=condition=ready pod -l app=connect -n "$NAMESPACE" --timeout=180s
kubectl wait --for=condition=ready pod -l app=scraper -n "$NAMESPACE" --timeout=120s
kubectl wait --for=condition=ready pod -l app=notification-service -n "$NAMESPACE" --timeout=120s

echo "Registering Debezium connector..."
CONNECT_POD=$(kubectl get pods -n "$NAMESPACE" -l app=connect -o jsonpath='{.items[0].metadata.name}')
kubectl cp connectors/debezium-postgres.json "$NAMESPACE/$CONNECT_POD:/tmp/connector.json"
kubectl exec -n "$NAMESPACE" "$CONNECT_POD" -- \
  curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d @/tmp/connector.json

echo "Deployment complete!"
echo "  Kafka UI: kubectl port-forward -n $NAMESPACE svc/kafkaui 8085:8085"
echo "  Prometheus: kubectl port-forward -n $NAMESPACE svc/prometheus 9090:9090"
echo "  Grafana: kubectl port-forward -n $NAMESPACE svc/grafana 3000:3000"
