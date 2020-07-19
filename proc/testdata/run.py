def run_basic(attr, t):
    return run("uname").output + run("echo", "ye").output.strip()

def check_no_write(attr, t):
    return run("touch", "aa").exit_code

def wd(attr, t):
    return run("pwd").output.strip()
