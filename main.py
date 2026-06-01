import time
import psutil
import sentry_sdk
from sentry_sdk import metrics
from config import SENTRY_DSN, INTERVAL_SECONDS

try:
    import docker
    _docker_available = True
except ImportError:
    _docker_available = False


def init_sentry():
    sentry_sdk.init(
        dsn=SENTRY_DSN,
        traces_sample_rate=0.0,
    )


def collect_host_metrics():
    cpu = psutil.cpu_percent(interval=1)
    mem = psutil.virtual_memory()
    disk = psutil.disk_usage("/")
    net = psutil.net_io_counters()

    metrics.gauge("host.cpu.percent", cpu, tags={"host": "macbook"})
    metrics.gauge("host.memory.used_mb", mem.used / 1024 / 1024, tags={"host": "macbook"})
    metrics.gauge("host.memory.percent", mem.percent, tags={"host": "macbook"})
    metrics.gauge("host.disk.used_gb", disk.used / 1024 / 1024 / 1024, tags={"host": "macbook"})
    metrics.gauge("host.disk.percent", disk.percent, tags={"host": "macbook"})
    metrics.gauge("host.net.bytes_sent_mb", net.bytes_sent / 1024 / 1024, tags={"host": "macbook"})
    metrics.gauge("host.net.bytes_recv_mb", net.bytes_recv / 1024 / 1024, tags={"host": "macbook"})

    print(f"  [host] cpu={cpu}%  mem={mem.percent}%  disk={disk.percent}%")


def collect_docker_metrics():
    if not _docker_available:
        return

    try:
        client = docker.from_env()
        containers = client.containers.list()
    except Exception as e:
        print(f"  [docker] unavailable: {e}")
        return

    for container in containers:
        try:
            stats = container.stats(stream=False)
            name = container.name

            # CPU %
            cpu_delta = stats["cpu_stats"]["cpu_usage"]["total_usage"] - stats["precpu_stats"]["cpu_usage"]["total_usage"]
            sys_delta = stats["cpu_stats"]["system_cpu_usage"] - stats["precpu_stats"]["system_cpu_usage"]
            num_cpus = stats["cpu_stats"].get("online_cpus", 1)
            cpu_pct = (cpu_delta / sys_delta) * num_cpus * 100.0 if sys_delta > 0 else 0.0

            # Memory
            mem_usage = stats["memory_stats"].get("usage", 0) / 1024 / 1024
            mem_limit = stats["memory_stats"].get("limit", 1) / 1024 / 1024
            mem_pct = (mem_usage / mem_limit) * 100 if mem_limit > 0 else 0.0

            tags = {"host": "macbook", "container": name}
            metrics.gauge("docker.cpu.percent", cpu_pct, tags=tags)
            metrics.gauge("docker.memory.used_mb", mem_usage, tags=tags)
            metrics.gauge("docker.memory.percent", mem_pct, tags=tags)

            print(f"  [docker] {name}  cpu={cpu_pct:.1f}%  mem={mem_usage:.1f}MB ({mem_pct:.1f}%)")

        except Exception as e:
            print(f"  [docker] error reading {container.name}: {e}")


def main():
    init_sentry()
    print(f"Starting infrastructure monitor — emitting every {INTERVAL_SECONDS}s")
    print(f"Docker SDK: {'available' if _docker_available else 'not installed'}\n")

    while True:
        print(f"[{time.strftime('%H:%M:%S')}] collecting metrics...")
        collect_host_metrics()
        collect_docker_metrics()
        print()
        time.sleep(INTERVAL_SECONDS)


if __name__ == "__main__":
    main()
