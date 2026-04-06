from setuptools import setup, find_packages

setup(
    name="applad",
    version="0.1.0",
    description="Applad Python Server SDK",
    author="Mittolabs",
    url="https://github.com/mittolabs/applad",
    packages=find_packages(),
    python_requires=">=3.8",
    install_requires=[],
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: BSD License",
        "Operating System :: OS Independent",
    ],
)
