

from langchain_openai import ChatOpenAI
from langchain_core.messages import HumanMessage,SystemMessage
from langchain_core.output_parsers import StrOutputParser

model = ChatOpenAI(model="gpt-3.5-turbo")

messages = [
    SystemMessage(content="Translate the following from English into Chinese."),
    HumanMessage(content="In this guide we will build an application to translate user input from one language to another.")
]

result = model.invoke(messages)

parser = StrOutputParser()
print(parser.invoke(result))


chain = model | parser
print(chain.invoke(messages))