import importlib.util
import tempfile
import unittest
from pathlib import Path


def load_serve_app():
    spec = importlib.util.spec_from_file_location("serve_app", Path(__file__).with_name("serve_app.py"))
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


def load_serve_spa():
    spec = importlib.util.spec_from_file_location("serve_spa", Path(__file__).with_name("serve_spa.py"))
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


class ServeAppStaticAssetTest(unittest.TestCase):
    def test_resolves_percent_encoded_static_asset_paths_before_spa_fallback(self):
        serve_app = load_serve_app()
        with tempfile.TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            card = root / "德扑完整牌组_已重命名_透明PNG" / "cards" / "6D.png"
            card.parent.mkdir(parents=True)
            card.write_bytes(b"png")
            (root / "index.html").write_text("<html></html>", encoding="utf-8")

            resolved = serve_app.resolve_spa_file(
                "/%E5%BE%B7%E6%89%91%E5%AE%8C%E6%95%B4%E7%89%8C%E7%BB%84_%E5%B7%B2%E9%87%8D%E5%91%BD%E5%90%8D_%E9%80%8F%E6%98%8EPNG/cards/6D.png",
                root=root,
            )

            self.assertEqual(resolved, card)


class ServeSpaStaticAssetTest(unittest.TestCase):
    def test_resolves_percent_encoded_static_asset_paths_before_spa_fallback(self):
        serve_spa = load_serve_spa()
        with tempfile.TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            card = root / "德扑完整牌组_已重命名_透明PNG" / "cards" / "6D.png"
            card.parent.mkdir(parents=True)
            card.write_bytes(b"png")
            (root / "index.html").write_text("<html></html>", encoding="utf-8")

            resolved = serve_spa.resolve_spa_file(
                "/%E5%BE%B7%E6%89%91%E5%AE%8C%E6%95%B4%E7%89%8C%E7%BB%84_%E5%B7%B2%E9%87%8D%E5%91%BD%E5%90%8D_%E9%80%8F%E6%98%8EPNG/cards/6D.png",
                root=root,
            )

            self.assertEqual(resolved, card)


if __name__ == "__main__":
    unittest.main()
