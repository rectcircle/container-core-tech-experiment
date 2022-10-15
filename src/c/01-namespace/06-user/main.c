// sudo apt install -y libcap2-bin
// gcc src/c/01-namespace/06-user/main.c && sudo setcap CAP_SETUID,CAP_SETGID,CAP_SETFCAP,CAP_DAC_OVERRIDE+ep a.out  && ./a.out
// sudo getcap a.out

#define _GNU_SOURCE	     // Required for enabling clone(2)
#include <sys/wait.h>    // For waitpid(2)
#include <sys/mount.h>   // For mount(2)
#include <sys/mman.h>    // For mmap(2)

#include <sched.h>	   // For clone(2)
#include <stdio.h>	   // For perror(3), printf(3), perror(3)
#include <unistd.h>    // For execv(3), sleep(3), read(2)
#include <stdlib.h>	   // For exit(3), system(3), free(3), realloc(3)
#include <errno.h>	   // For errno(3), strerror(3)
#include <string.h>	   // For strtok(3)
#include <fcntl.h>     // For open(2)

#define errExit(msg)    do { perror(msg); exit(EXIT_FAILURE); \
							   } while (0)

#define STACK_SIZE (1024 * 1024)

char *testFileName = "testFile";

// https://stackoverflow.com/a/44894946
/* Size of each input chunk to be
   read and allocate for. */

#define  READALL_CHUNK  4096
#define  READALL_OK          0  /* Success */
#define  READALL_INVALID    -1  /* Invalid parameters */
#define  READALL_ERROR      -2  /* Stream error */
#define  READALL_TOOMUCH    -3  /* Too much input */
#define  READALL_NOMEM      -4  /* Out of memory */

/* This function returns one of the READALL_ constants above.
   If the return value is zero == READALL_OK, then:
	 (*dataptr) points to a dynamically allocated buffer, with
	 (*sizeptr) chars read from the file.
	 The buffer is allocated for one extra char, which is NUL,
	 and automatically appended after the data.
   Initial values of (*dataptr) and (*sizeptr) are ignored.
*/
int readall(FILE *in, char **dataptr, size_t *sizeptr)
{
	char  *data = NULL, *temp;
	size_t size = 0;
	size_t used = 0;
	size_t n;

	/* None of the parameters can be NULL. */
	if (in == NULL || dataptr == NULL || sizeptr == NULL)
		return READALL_INVALID;

	/* A read error already occurred? */
	if (ferror(in))
		return READALL_ERROR;

	while (1) {

		if (used + READALL_CHUNK + 1 > size) {
			size = used + READALL_CHUNK + 1;

			/* Overflow check. Some ANSI C compilers
			   may optimize this away, though. */
			if (size <= used) {
				free(data);
				return READALL_TOOMUCH;
			}

			temp = realloc(data, size);
			if (temp == NULL) {
				free(data);
				return READALL_NOMEM;
			}
			data = temp;
		}

		n = fread(data + used, 1, READALL_CHUNK, in);
		if (n == 0)
			break;

		used += n;
	}

	if (ferror(in)) {
		free(data);
		return READALL_ERROR;
	}

	temp = realloc(data, used + 1);
	if (temp == NULL) {
		free(data);
		return READALL_NOMEM;
	}
	data = temp;
	data[used] = '\0';

	*dataptr = data;
	*sizeptr = used;

	return READALL_OK;
}



void print_caps() {
	FILE *f = fopen("/proc/self/status", "r");
	if (f == NULL)
		errExit("fopen");
	char *buf;
	size_t len;
	if (readall(f, &buf, &len) != READALL_OK)
		errExit("readall");
	fclose(f);

	char *delimiter = "\r\n";
	char *line = strtok(buf, delimiter);
	while (line != NULL) {
		char *pre = "Cap";
		if (strncmp(pre, line, strlen(pre)) == 0)
			printf("%s\n", line);
		line = strtok(NULL, delimiter);
	}
}

static void
update_map(char *mapping, char *map_file)
{
	int fd, j;
	size_t map_len = map_len = strlen(mapping);

	fd = open(map_file, O_RDWR);
	if (fd == -1)
	{
		fprintf(stderr, "open %s: %s\n", map_file, strerror(errno));
		exit(EXIT_FAILURE);
	}
	if (write(fd, mapping, map_len) != map_len)
	{
		fprintf(stderr, "write %s: %s\n", map_file, strerror(errno));
		exit(EXIT_FAILURE);
	}
	close(fd);
}

struct child_args {
	int pipe_fd[2]; /* Pipe used to synchronize parent and child */
};

char *const test_scripts[] = {
	"/bin/bash",
	"-c",
	"echo '>>>' 01.当前进程ID && echo $$ && echo \
	&& echo '>>>' 02.查看当前进程 Caps && cat /proc/self/status | grep Cap && echo \
	&& echo '>>>' 03.当前进程身份 && id && echo \
	&& echo '>>>' 04.执行 ps -ef && ps -ef && echo \
	&& echo '>>>' 05.执行 ls -al / && ls -al / && echo \
	&& echo '>>>' 06.执行 ls -al && ls -al && echo \
	&& echo '>>>' 07.执行 ls -al && ls -al && echo \
	&& echo '>>>' 08.写入 abc 到 testFile 并查看 && echo 'abc' > testFile && cat testFile && echo \
	&& echo '>>>' 09.sudo 更改 testFile owner 为 root && sudo chown root:root testFile && ls -al testFile && echo \
	",
	NULL};

int new_namespace_func(void *args) {
	struct child_args *typedArgs = (struct child_args *)args;

	char ch;
	close(typedArgs->pipe_fd[1]);
	if (read(typedArgs->pipe_fd[0], &ch, 1) != 0) {
        fprintf(stderr, "Failure in child: read from pipe returned != 0\n");
        exit(EXIT_FAILURE);
    }

	printf("时序 05: 打印进程 B 的 Caps、 进程 ID 和 用户 ID\n");
	print_caps();
	printf("pid: %d\n", getpid());
	printf("uid: %d\n", getuid());
	printf("\n");

	printf("时序 06: 尝试更改测试文件 owner\n");
	if (chown(testFileName, 0, 0) < 0)
		errExit("chown-root");
	if (chown(testFileName, getuid(), getuid()) < 0)
		errExit("chown-uid");
	printf("成功\n\n");

	printf("时序 07: 重新挂载 /proc\n\n");
	if (mount(NULL, "/", NULL, MS_SLAVE | MS_REC, NULL) == -1) // 阻止挂载事件传播到其他 Mount Namespace
		errExit("mount-MS_SLAVE");
	if (mount("proc", "/proc", "proc", 0, NULL) == -1)
		errExit("mount-proc");

	printf("时序 08: 执行测试脚本\n");
	execv(test_scripts[0], test_scripts);

	return 0;
}

int main(int argc, char *argv[]) {

	printf("时序 01: 打印进程 A 的 Caps 和 进程 ID\n");
	print_caps();
	printf("pid: %d\n", getpid());
	printf("\n");

	printf("时序 02: 创建一个测试文件\n\n");
	int f = open(testFileName, O_WRONLY | O_CREAT | O_TRUNC, 0644);
	if (f < 0)
		errExit("open-testFile");

	printf("时序 03: 创建一个新进程 B，这个进程位于新的 User、Mount、PID Namespace\n");
	struct child_args args;
	if ( pipe(args.pipe_fd) == -1)
		errExit("pipe");
	void *child_stack = mmap(NULL, STACK_SIZE,
							 PROT_READ | PROT_WRITE,
							 MAP_PRIVATE | MAP_ANONYMOUS | MAP_STACK,
							 -1, 0);
	if (child_stack == MAP_FAILED)
		errExit("mmap");
	pid_t pid = clone(new_namespace_func, child_stack + STACK_SIZE, SIGCHLD | CLONE_NEWUSER | CLONE_NEWPID | CLONE_NEWNS, &args);
	if (pid < 0)
		errExit("clone");
	printf("pid: %d\n\n", getpid());

	printf("时序 04: 配置子进程的 id map\n\n");

	char map_path[128];
	sprintf(map_path, "/proc/%d/uid_map", pid);
	update_map("0 0 4294967295", map_path);
	sprintf(map_path, "/proc/%d/gid_map", pid);
	update_map("0 0 4294967295", map_path);
	close(args.pipe_fd[1]);

	if (waitpid(pid, NULL, 0) < 0)
		errExit("pid");
	printf("时序 09: 子进程 B 退出，并清理现场\n\n");
	unlink(testFileName);
	return 0;
}
