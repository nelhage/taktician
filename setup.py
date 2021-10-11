#!/usr/bin/env python
from setuptools import setup, find_packages

setup(
    name='xformer',
    version='0.1.0',
    description='A toy transformer model',
    author='Nelson Elhage',
    author_email='nelhage@nelhage.com',
    packages=find_packages(exclude=('test',))
)
