# Release Procedure Checklist

- Update the application
- Tag the application commit
  - This builds new container images
- Update the `charts` directory
  - `values.yaml`
    - Update image tags if needed
  - `Chart.yaml`
    - Update chart and application versions
- Tag the helm commit
  - This packages and releases the helm charts