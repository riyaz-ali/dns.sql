from setuptools import setup

version = {}
with open("sqlite_dns/version.py") as fp:
    exec(fp.read(), version)
VERSION = version['__version__']

setup(
    name="sqlite-dns",
    description="A sqlite3 extension that allows you to query DNS using SQL",
    author="Riyaz Ali",
    url="https://github.com/riyaz-ali/dns.sql",
    project_urls={
        "Issues": "https://github.com/riyaz-ali/dns.sql/issues",
        "Changelog": "https://github.com/riyaz-ali/dns.sql/releases",
    },
    license="MIT License",
    version=VERSION,
    packages=["sqlite_dns"],
    package_data={"sqlite_dns": ['lib/*.so', 'lib/*.dylib']},
    install_requires=[],
    python_requires=">=3.7",
)
