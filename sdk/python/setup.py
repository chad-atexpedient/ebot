"""
Setup configuration for ebot Python SDK.
"""

from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="ebot",
    version="0.1.0",
    author="Expedient",
    author_email="support@expedient.com",
    description="Official Python SDK for the ebot platform",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/chad-atexpedient/ebot",
    packages=find_packages(),
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "Topic :: Internet :: WWW/HTTP",
    ],
    python_requires=">=3.8",
    install_requires=[
        "httpx>=0.24.0",
        "typing-extensions>=4.0.0",
    ],
    extras_require={
        "dev": [
            "pytest>=7.0.0",
            "pytest-asyncio>=0.21.0",
            "pytest-cov>=4.0.0",
            "black>=23.0.0",
            "mypy>=1.0.0",
            "ruff>=0.1.0",
        ],
    },
    project_urls={
        "Documentation": "https://docs.expedient.cloud/ebot",
        "Source": "https://github.com/chad-atexpedient/ebot",
        "Tracker": "https://github.com/chad-atexpedient/ebot/issues",
    },
)
