from setuptools import setup
from torch.utils import cpp_extension

setup(
  ext_modules=[cpp_extension.CppExtension('tak_ext', ['ext/tak.cpp'])],
  cmdclass={'build_ext': cpp_extension.BuildExtension}
)
