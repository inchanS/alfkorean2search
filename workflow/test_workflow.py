# coding=utf-8
import unittest
from unittest.mock import patch
import korean_search

class MyTestCase(unittest.TestCase):
    @patch('korean_search.get_data')
    def test_korean_search(self, mock_get_data):
        # 모의 데이터 설정
        mock_response = {
            'json': ["var dq_searchKeyword='사랑'; var dq_searchResultList=new Array('사랑','사랑하다');", 12345]
        }
        mock_get_data.return_value = mock_response

        res = korean_search.get_data('사랑')
        self.assertTrue(len(res['json']) > 0)
        # 추가적으로 parse_suggestions 함수를 테스트
        js_code = res['json'][0]
        suggestions = korean_search.parse_suggestions(js_code)
        self.assertIn('사랑', suggestions)

if __name__ == '__main__':
    unittest.main()
