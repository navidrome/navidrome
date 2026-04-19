import mlflow

uri = "http://129.114.27.248:8000"
mlflow.set_tracking_uri(uri)

try:
    with mlflow.start_run(run_name="test_connection"):
        mlflow.log_param("status", "working")
        print(f"Successfully started run: {mlflow.active_run().info.run_id}")
except Exception as e:
    print(f"Failed to connect to MLflow: {e}")