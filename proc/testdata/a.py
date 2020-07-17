def some_number(attr, t):
    return 42

def some_string(attr, t):
    return "1" + "." + "2"

def read_attr(attr, t):
    return "name={}, path={}".format(attr.name, attr.path)

def parent_info(attr, t):
    parent = attr.parent
    return "{}: name={}, path={}".format(parent.type, parent.name, parent.path)

def target_info(attr, t):
    return "{}: name={}, path={}, deps={}, details={}".format(t.type, t.name, t.path, t.deps, t.details)
