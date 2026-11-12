import unittest
from grouping import group_regions, recognized_word, chat_prefix, author_above_date, DATE

def line(text, x, y, width=120):
    return dict(text=text,x=x,y=y,width=width,height=18,score=.99)

class GroupingTests(unittest.TestCase):
    def test_continuation_and_message_boundary(self):
        lines=[line('alice: i try my',0,0),line('best',0,22,35),line('bob: hello',0,45)]
        result, mapping=group_regions(lines,[dict(label='text',coordinate=[0,0,300,100])])
        self.assertEqual([r['text'] for r in result],['alice: i try my best','bob: hello'])
        self.assertEqual(mapping[0]['members'],[0,1])
        self.assertEqual(len(result[0]['boxes']),2)

    def test_separate_blocks_and_missing_layout(self):
        lines=[line('i try my',0,0),line('best',0,22,35)]
        self.assertEqual(len(group_regions(lines,[])[0]),1)
        layout=[dict(label='text',coordinate=[0,0,140,19]),dict(label='text',coordinate=[0,21,140,42])]
        self.assertEqual(len(group_regions(lines,layout)[0]),2)

    def test_fragmented_chat_row_then_wrapped_word(self):
        lines=[line('i try my',232,573,52),line('16:26alice:',0,574,200),line('best',0,604,32)]
        result,_=group_regions(lines,[dict(label='text',coordinate=[0,550,300,650])])
        self.assertEqual([r['text'] for r in result],['16:26alice: i try my best'])

    def test_username_preserved_but_continuation_at_left_margin_is_merged(self):
        first=line('Im so hungry',183,39,85);first.update(message_start=True,flow_left=19)
        second=line('now',10,63,30)
        result,_=group_regions([first,second],[dict(label='text',coordinate=[180,38,280,60])])
        self.assertEqual([r['text'] for r in result],['Im so hungry now'])

    def test_recognized_punctuation_and_separate_closing_marker(self):
        self.assertTrue(recognized_word('"'))
        self.assertTrue(recognized_word('*'))
        self.assertFalse(recognized_word('🙂'))
        body=line('I have you on follow',0,0)
        star=line('*',70,23,8);star['erase']=[dict(x=72,y=25,width=2,color=[24,24,24])]
        result,mapping=group_regions([body,star],[])
        self.assertEqual(result[0]['text'],'I have you on follow')
        self.assertEqual(len(result[0]['boxes']),1)
        self.assertEqual(result[0]['erase'][0]['y'],25)
        self.assertEqual(mapping[0]['members'],[0,1])

    def test_article_author_date_title_and_tags_are_separate(self):
        regions=[line('FrancisTRDev',67,178,77),line('11月9日',68,195,40),line('How do you truly stand out in Tech?',73,225,485),line('#discuss',72,267,48),line('#community #productivity #career',139,267,232)]
        regions[2]['height']=31
        result,_=group_regions(regions,[])
        self.assertEqual(len(result),5)
        self.assertEqual(result[2]['text'],'How do you truly stand out in Tech?')
        self.assertEqual(result[2]['height'],31)

    def test_chat_continuation_tolerates_ocr_line_height_variation(self):
        first=line('kromede? nice, i might give',110,0,202)
        first.update(height=19,message_start=True,flow_left=0)
        second=line('it a try next week on TW, haha',0,24,270)
        second['height']=14
        third=line('nextuser: another message',100,50,190)
        third['message_start']=True
        result,mapping=group_regions([first,second,third],[])
        self.assertEqual(len(result),2)
        self.assertEqual(mapping[0]['members'],[0,1])
        self.assertIn('next week on TW',result[0]['text'])
        self.assertEqual(len(result[0]['boxes']),2)

    def test_title_colon_is_not_username_and_spaced_author_is_preserved(self):
        title='A Small Laptop Is a Full Build Kit: Bootstrapping with a'
        self.assertIsNone(chat_prefix(title))
        self.assertIsNotNone(chat_prefix('22:45 user_name: hello'))
        self.assertIsNone(chat_prefix('Warning: Important',[0,0,300,30],[dict(label='doc_title',coordinate=[0,0,300,30])]))
        self.assertTrue(author_above_date('Earl Grey',[60,65,125,82],[[60,84,105,98]]))
        self.assertFalse(author_above_date('A Small Laptop',[60,110,300,141],[[60,145,105,159]]))
        self.assertTrue(DATE.fullmatch('11月1日'))

    def test_separate_columns(self):
        lines=[line('left',0,0),line('right',500,0)]
        self.assertEqual(len(group_regions(lines,[dict(label='text',coordinate=[0,0,800,100])])[0]),2)

if __name__ == '__main__': unittest.main()
