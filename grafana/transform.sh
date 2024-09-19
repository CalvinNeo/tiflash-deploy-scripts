sudo sed -i 's/7.5.17/7.5.11/' metrics/grafana/tiflash_summary.json
sudo sed -i 's/DS_CALVIN-TEST/DS_TEST-CLUSTER/' metrics/grafana/tiflash_summary.json
sudo sed -i 's/calvin-test/Test-Cluster/' metrics/grafana/tiflash_summary.json

sudo sed -i 's/7.5.17/7.5.11/' metrics/grafana/tiflash_summary.json
sudo sed -i 's/Test-Cluster-79-TiFlash-Summary/Test-Cluster-TiFlash-Summary/' metrics/grafana/tiflash_summary.json
sudo sed -i 's/DS_CALVIN-TEST-79/DS_TEST-CLUSTER/' metrics/grafana/tiflash_summary.json
sudo sed -i 's/calvin-test-79/Test-Cluster/' metrics/grafana/tiflash_summary.json
