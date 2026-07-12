"""Deprecated CABLE profiling API.

Use :mod:`bridge_py.truss.profile` in new code.
"""
from ..truss.profile import QueryProfile, profile_query

__all__ = ["QueryProfile", "profile_query"]
