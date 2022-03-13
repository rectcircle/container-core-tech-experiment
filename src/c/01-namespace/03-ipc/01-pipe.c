// gcc src/c/01-namespace/03-ipc/01-pipe.c && sudo ./a.out
#include <unistd.h>    // For pipe(2), STDOUT_FILENO
#include <limits.h>    // For PIPE_BUF
#include <stdlib.h>   // For EXIT_FAILURE, exit
#include <stdio.h>    // For perror
#include <sys/wait.h>  // For waitpid(2)
#include <string.h>    // For strlen(3)
#include <sys/msg.h>  // For waitpid(2)

#define errExit(msg)    do { perror(msg); exit(EXIT_FAILURE); \
                               } while (0)

void main()
{
    int n;
    int fd[2];
    pid_t pid;
    const char *msg = "hello world\n";
    const int MAXLINE = 1024;
    char line[MAXLINE];

    if (pipe(fd) < 0)
        errExit("pipe");

    if ((pid = fork())< 0)
        errExit("fork");

    if (pid > 0) // 父进程
    {
        close(fd[0]); // 父进程不需要使用管道的读取端点，所以关闭它
        write(fd[1], msg, strlen(msg));
        wait(NULL);
    }
    else // 子进程
    {
        close(fd[1]); // 子进程不需要使用管道的写入端点，所以关闭它
        n = read(fd[0], line, MAXLINE);
        write(STDOUT_FILENO, line, n);
    }
}
