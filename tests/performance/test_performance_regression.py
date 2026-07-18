import importlib.util, json, tempfile, unittest
from pathlib import Path

ROOT=Path(__file__).resolve().parents[2]
spec=importlib.util.spec_from_file_location('pr', ROOT/'tasks/performance_regression.py')
pr=importlib.util.module_from_spec(spec); spec.loader.exec_module(pr)

class PerformanceRegressionTest(unittest.TestCase):
    def test_allowed_boundary(self):
        self.assertTrue(pr.allowed(135,100,.35))
        self.assertFalse(pr.allowed(135.01,100,.35))
    def test_key(self):
        self.assertEqual(pr.key_for({'scenario_id':'s','algorithm':'a','query_id':'q'}),'s/a/q')

    def test_tier_policy(self):
        self.assertEqual(pr.policy_for('small')['minimum_sample_count'], 60)
        self.assertEqual(pr.policy_for('medium')['minimum_sample_count'], 10)
        self.assertEqual(pr.policy_for('large')['minimum_sample_count'], 3)
        self.assertLess(pr.policy_for('large')['performance']['solver_time_p50_max_increase_ratio'], pr.policy_for('small')['performance']['solver_time_p50_max_increase_ratio'])

    def test_unknown_tier(self):
        with self.assertRaises(ValueError):
            pr.policy_for('unknown')

if __name__=='__main__': unittest.main()
