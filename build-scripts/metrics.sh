generate_image_metrics() {
  if [ -z "${CI_REGISTRY_IMAGE}" ]; then
    echo "CI_REGISTRY_IMAGE not set, can not inspect image metrics"
    exit 1
  fi

  mkdir -p metrics
  METRICS_FILE=metrics/image_metrics.txt
  cat << EOF > "${METRICS_FILE}"
# TYPE container_total_compressed_bytes gauge
# UNIT container_total_compressed_bytes bytes
# HELP container_total_compressed_bytes Sum of container layer sizes in bytes
# TYPE container_layers gauge
# UNIT container_layers number
# HELP container_layers Number of layers in container
EOF

  # iterate over all images with entries in 'artifacts/final/*'
  for image in artifacts/final/* ; do
    # fetch data with skopeo
    tagged_image="$(cat "${image}")"
    skopeo inspect --raw "docker://${CI_REGISTRY_IMAGE}/${tagged_image}" > data.json

    # collect desired data
    layers=$(jq -r '.layers | length' data.json)
    total_size=$(jq -r '[ .layers[].size ] | add' data.json)

    # append to METRICS_FILE
    cat << EOF >> "${METRICS_FILE}"
container_layers{image="${tagged_image}"} ${layers}
container_total_compressed_bytes{image="${tagged_image}"} ${total_size}
EOF

  done

  # terminate the file according to specification. '# EOF\n'
  echo '# EOF' >> "${METRICS_FILE}"
}
