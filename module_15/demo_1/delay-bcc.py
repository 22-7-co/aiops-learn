# 从 bcc 模块中导入 BPF 类，用于编写和加载 eBPF 程序
from bcc import BPF

# 从 time 模块导入 sleep 用于暂停，我们会用它控制程序的执行节奏
from time import sleep

# 导入 Python 原生的 time 模块，用于格式化时间戳
import time

# 导入 argparse 模块，用于解析命令行参数
import argparse

# 构造一个命令行参数解析器，用于接受用户输入的目标进程 PID
parser = argparse.ArgumentParser(
    description="Trace HTTP request latency for a specific process."
)
# 添加 '-p' 或 '--pid' 参数，接受整数类型的 PID，且参数是必需的
parser.add_argument(
    "-p", "--pid", type=int, required=True, help="The PID of the Go process to trace"
)
# 从命令行解析用户输入的参数
args = parser.parse_args()

# 将解析出的 PID 存储在变量 target_pid 中
target_pid = args.pid

# 定义 eBPF 程序，写在一个多行的字符串中。这段代码运行在 Linux 内核中。
# 使用 eBPF 能够高效地抓取、过滤和分析内核级别的事件数据
bpf_text = """
#include <uapi/linux/ptrace.h>  // 包含 eBPF 程序需要的用户态跟踪头文件
#include <linux/sched.h>         // 提供调度程序相关的结构体和定义

// 定义一个结构体用于存储跟踪的数据，它将从内核发送到用户态
struct data_t {
    u32 pid;                      // 进程 ID
    u64 latency;                  // 延迟时间
    char comm[TASK_COMM_LEN];     // 进程名
};

// 定义一个哈希表用于存储每个 PID 的开始时间戳
BPF_HASH(start, u32);
// 定义一个事件缓冲区，用于将事件从内核传递到用户态
BPF_PERF_OUTPUT(events);

// 定义一个 eBPF 函数，触发点是 tcp_sendmsg（表示发送 TCP 消息的时刻）
int trace_start(struct pt_regs *ctx, struct sock *sk) {
    // 获取当前进程的 PID，高 32 位存储的是 PID
    u32 pid = bpf_get_current_pid_tgid() >> 32;

    // 只跟踪用户指定的目标 PID 事件，非目标 PID 就直接返回
    if (pid != TARGET_PID) { 
        return 0;
    }

    // 获取内核的高精度时间戳（单位是 ns）
    u64 ts = bpf_ktime_get_ns();
    // 将时间戳存入哈希表（start）中，键为 PID，值为时间戳
    start.update(&pid, &ts);

    return 0;  // 返回 0 代表函数成功结束
}

// 定义另一个 eBPF 函数，触发点是 tcp_cleanup_rbuf（表示接收 TCP 消息完成的时刻）
int trace_end(struct pt_regs *ctx, struct sock *sk) {
    // 获取当前进程的 PID
    u32 pid = bpf_get_current_pid_tgid() >> 32;

    // 只跟踪用户指定的目标 PID
    if (pid != TARGET_PID) { 
        return 0;
    }

    // 从哈希表中查找之前记录的开始时间戳
    u64 *tsp = start.lookup(&pid);
    // 如果没有找到开始时间戳，表示有可能错过了 start 事件，直接返回
    if (tsp == 0) {
        return 0;
    }

    // 计算延迟：当前时间戳减去开始时间戳
    u64 delta = bpf_ktime_get_ns() - *tsp;
    // 删除哈希表中的记录，因为数据已经用完了
    start.delete(&pid);

    // 初始化一个数据结构，用于存储并发送事件
    struct data_t data = {};
    data.pid = pid;                 // 存储当前进程的 PID
    data.latency = delta;           // 保存延迟时间
    bpf_get_current_comm(&data.comm, sizeof(data.comm));  // 获取当前进程名

    // 将收集到的数据发送到用户态
    events.perf_submit(ctx, &data, sizeof(data));
    return 0;  // 返回 0 代表函数成功结束
}
"""

# 替换 eBPF 程序中的占位符 TARGET_PID 为用户输入的 PID
bpf_text = bpf_text.replace("TARGET_PID", str(target_pid))

# 使用用户编写的 eBPF 程序 text 来初始化 BPF 对象
b = BPF(text=bpf_text)

# 将 trace_start 函数附加到 `tcp_sendmsg` 内核函数上，当函数被调用时会触发 trace_start
b.attach_kprobe(event="tcp_sendmsg", fn_name="trace_start")
# 将 trace_end 函数附加到 `tcp_cleanup_rbuf` 内核函数上，当函数被调用时会触发 trace_end
b.attach_kprobe(event="tcp_cleanup_rbuf", fn_name="trace_end")


# 定义一个回调函数，用于处理从 eBPF 程序发送到用户态的事件数据
def print_event(cpu, data, size):
    # 从 eBPF 传过来的二进制数据解析成 data 结构体
    event = b["events"].event(data)
    # 格式化打印事件的数据，包括时间、进程号、进程名和延迟
    print(
        f"[{time.strftime('%H:%M:%S')}] PID: {event.pid}, COMM: {event.comm.decode('utf-8', 'replace')}, Latency: {event.latency / 1e6:.2f} ms"
    )


# 将 eBPF 的事件缓冲区绑定到用户态的回调函数上，便于处理数据
b["events"].open_perf_buffer(print_event)

# 输出提示信息，告诉用户当前正在跟踪哪个进程
print(f"Tracing HTTP request latency for PID {target_pid}... Hit Ctrl-C to end.")

# 进入一个无限循环，不断从事件缓冲区中读取和处理数据
while True:
    try:
        # 阻塞等待数据事件，并调用上面定义的 print_event 函数处理事件
        b.perf_buffer_poll()
    except KeyboardInterrupt:
        # 捕获 Ctrl-C 信号，退出程序
        exit()
