# coding=utf-8
import unittest
from unittest.mock import patch
import korean_search

class MyTestCase(unittest.TestCase):
    @patch('korean_search.get_data')
    def test_korean_search(self, mock_get_data):
        # 1. 모의(Mock) 데이터 설정: 표준국어대사전 OpenAPI JSON 구조에 맞게 변경
        mock_response = {
            'channel': {
                'item': [
                    {
                        'word': '사랑',
                        'sup_no': '1',
                        'target_code': '12345',
                        'link': '',
                        'sense': [
                            {'definition': '어떤 사람이나 존재를 몹시 아끼고 귀중히 여기는 마음.'}
                        ]
                    }
                ]
            }
        }
        mock_get_data.return_value = mock_response

        # 2. get_data 호출 시 테스트용 가짜 API 키(dummy_api_key)를 두 번째 인자로 전달
        res = korean_search.get_data('사랑', 'dummy_api_key')
        self.assertIn('channel', res)

        # 3. parse_suggestions 함수 테스트 (이제 응답 딕셔너리 전체를 넘겨야 함)
        suggestions = korean_search.parse_suggestions(res)

        # 4. 반환값이 List[Dict] 형태이므로 데이터 검증 방식 변경
        self.assertTrue(len(suggestions) > 0)
        self.assertEqual(suggestions[0]['word'], '사랑(1)')
        self.assertIn('아끼고 귀중히 여기는', suggestions[0]['definition'])

if __name__ == '__main__':
    unittest.main()