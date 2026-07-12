"""Backward-compatible graph imports.

New code should import from :mod:`bridge_py.core.graph`.
"""
from .core.graph import *  # noqa: F401,F403
