from sklearn.preprocessing import StandardScaler
from sklearn.linear_model import LogisticRegression
from sklearn.pipeline import make_pipeline
from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
from sklearn.metrics import accuracy_score
from sklearn.pipeline import Pipeline

pipe = make_pipeline(
    StandardScaler(),
    LogisticRegression(random_state=0)
)

X,y = load_iris(return_X_y=True)

X_train, X_test, y_train, y_test = train_test_split(X,y,random_state=0)

pipe.fit(X_train, y_train)

Pipeline(steps=[('standardscaler', StandardScaler()),
                ('logisticregression', LogisticRegression(random_state=0))])

print(accuracy_score(pipe.predict(X_test), y_test))