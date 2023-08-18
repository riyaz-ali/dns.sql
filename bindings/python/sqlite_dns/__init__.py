#  dns.sql - sqlite3 extension that allows querying dns using sql

def path():
    """
    path() returns the local path to the correct dynamic library to load using sqlite3_load_extension() function
    """
    import os

    loadable_path = os.path.join(os.path.dirname(__file__), 'lib', 'sqlite_dns')
    return os.path.normpath(loadable_path)
